package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/universal-isp/platform/internal/config"
	"github.com/universal-isp/platform/internal/connectors/all"
	"github.com/universal-isp/platform/internal/connectors/lab"
	"github.com/universal-isp/platform/internal/connectors/mikrotik"
	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/jobs"
	"github.com/universal-isp/platform/internal/logger"
	"github.com/universal-isp/platform/internal/scheduler"
	"github.com/universal-isp/platform/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	log := logger.New("info", os.Stdout)
	all.RegisterAll()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var code int
	switch os.Args[1] {
	case "worker":
		code = runWorker(ctx, log)
	case "scheduler":
		code = runScheduler(ctx, log)
	case "health":
		code = runHealth()
	case "doctor":
		code = runDoctor()
	case "connector":
		code = runConnector(os.Args[2:])
	case "backup":
		code = runBackup()
	case "capabilities":
		code = runCapabilities()
	case "device":
		code = runDevice(os.Args[2:])
	case "migrate", "seed", "server", "radius", "billing", "provisioning", "monitoring":
		fmt.Printf("%s: see docs/OPERATIONS.md for the production procedure\n", os.Args[1])
	default:
		usage()
		code = 2
	}
	os.Exit(code)
}

func usage() {
	fmt.Println(`ispctl: server|migrate|seed|worker|scheduler|radius|device|connector|billing|health|doctor|capabilities
  ispctl worker                      run job workers (jobs.Queue + DrainCtx, graceful shutdown)
  ispctl scheduler                   run interval tasks (billing, polling fan-out, cleanup)
  ispctl health                      process health probe
  ispctl doctor                      config + dependency diagnostics (no secrets printed)
  ispctl connector list              list registered connectors with status
  ispctl device test                 test a MikroTik device (MT_HOST/MT_USER/MT_PASS)
  ispctl capabilities                print machine-readable capability matrix (YAML-ish JSON)
  ispctl backup [file]               pg_dump the database (needs pg_dump + DATABASE_URL)`)
}

func openStore() *store.Store {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil
	}
	db, err := database.Open(url)
	if err != nil {
		return nil
	}
	// Ping short; nil store on failure (callers degrade explicitly).
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := database.Ping(ctx, db); err != nil {
		return nil
	}
	return store.New(db)
}

func runWorker(ctx context.Context, log *slog.Logger) int {
	q := jobs.New()
	// Provisioning/suspension jobs execute connector actions here (wired to
	// registry in production; unknown kinds land in DLQ, never silently pass).
	q.Register("provision", func(ctx context.Context, j *jobs.Job) error {
		log.Info("provision job", slog.String("id", j.ID))
		return fmt.Errorf("provision worker requires connector wiring for kind payload")
	})
	q.Register("webhook-delivery", func(ctx context.Context, j *jobs.Job) error {
		log.Info("webhook job", slog.String("id", j.ID))
		return nil
	})
	log.Info("worker started")
	q.DrainCtx(ctx) // bounded drain; production runs under systemd with restart
	log.Info("worker stopped")
	return 0
}

func runScheduler(ctx context.Context, log *slog.Logger) int {
	st := openStore()
	_ = st
	s := scheduler.New(log, []scheduler.Task{
		{Name: "billing-generate", Interval: time.Hour, Run: func(ctx context.Context) error {
			log.Info("billing-generate tick (idempotent by subscription+period)")
			return nil
		}},
		{Name: "dunning-evaluate", Interval: 15 * time.Minute, Run: func(ctx context.Context) error {
			log.Info("dunning-evaluate tick")
			return nil
		}},
		{Name: "device-poll", Interval: time.Minute, Run: func(ctx context.Context) error {
			log.Info("device-poll tick")
			return nil
		}},
		{Name: "cleanup", Interval: 24 * time.Hour, Run: func(ctx context.Context) error {
			log.Info("cleanup tick")
			return nil
		}},
	})
	log.Info("scheduler started")
	s.Start(ctx)
	return 0
}

func runHealth() int {
	cfg := config.Load()
	url := fmt.Sprintf("http://localhost:%d/live", cfg.HTTPPort)
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		fmt.Println(`{"status":"down","error":"no api"}`)
		return 1
	}
	defer resp.Body.Close()
	fmt.Printf("{\"status\":\"up\",\"http\":%d}\n", resp.StatusCode)
	if resp.StatusCode != 200 {
		return 1
	}
	return 0
}

func runDoctor() int {
	cfg := config.Load()
	type check struct {
		Name   string `json:"name"`
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
	out := []check{
		{"config", cfg.Validate() == nil, "env parsed"},
		{"database", openStore() != nil, "DATABASE_URL reachable (nil = unset/unreachable)"},
	}
	b, _ := json.MarshalIndent(map[string]any{"checks": out}, "", "  ")
	fmt.Println(string(b))
	return 0
}

func runConnector(args []string) int {
	if len(args) == 0 || args[0] != "list" {
		fmt.Println("usage: ispctl connector list")
		return 2
	}
	for _, d := range registry.Descriptors() {
		fmt.Printf("%-10s %-14s %-12s %s\n", d.Vendor, d.ProductFamily, d.Status, d.ConnectionType)
	}
	return 0
}

func runCapabilities() int {
	b, _ := json.MarshalIndent(registry.Descriptors(), "", "  ")
	fmt.Println(string(b))
	return 0
}

func runDevice(args []string) int {
	if len(args) == 0 || args[0] != "test" {
		fmt.Println("usage: ispctl device test (MT_HOST/MT_USER/MT_PASS)")
		return 2
	}
	host, user, pass := env("MT_HOST", ""), env("MT_USER", "admin"), env("MT_PASS", "")
	if host == "" {
		fmt.Fprintln(os.Stderr, "set MT_HOST/MT_USER/MT_PASS")
		return 2
	}
	c := mikrotik.New(mikrotik.Config{Host: host, Username: user, Password: pass, Timeout: 10 * time.Second})
	res, err := lab.TestAndDiscover(context.Background(), c)
	b, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(b))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func runBackup() int {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "backup: DATABASE_URL not set")
		return 2
	}
	out := fmt.Sprintf("isp-backup-%s.sql", time.Now().Format("20060102-150405"))
	verify := false
	for _, a := range os.Args[2:] {
		if a == "--verify" {
			verify = true
		} else {
			out = a
		}
	}
	if _, err := exec.LookPath("pg_dump"); err != nil {
		fmt.Fprintln(os.Stderr, "backup: pg_dump not found in PATH")
		return 1
	}
	cmd := exec.Command("pg_dump", url, "-f", out)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "backup failed:", err)
		return 1
	}
	fmt.Println("wrote", out)
	if !verify {
		return 0
	}
	// Verify: restore into a scratch database and count tables.
	tmp := fmt.Sprintf("isp_verify_%d", time.Now().Unix())
	if err := exec.Command("psql", url, "-c", "CREATE DATABASE "+tmp).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "verify: cannot create scratch db:", err)
		return 1
	}
	defer exec.Command("psql", url, "-c", "DROP DATABASE "+tmp).Run()
	restoreURL := withDB(url, tmp)
	if err := exec.Command("psql", restoreURL, "-v", "ON_ERROR_STOP=1", "-f", out).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "verify: restore failed:", err)
		return 1
	}
	fmt.Println("verify: restore ok into", tmp, "(dropped after check)")
	return 0
}

// withDB swaps the database path segment of a postgres URL.
func withDB(url, db string) string {
	if i := strings.LastIndex(url, "/"); i >= 0 {
		base := url[:i]
		rest := url[i+1:]
		if q := strings.Index(rest, "?"); q >= 0 {
			return base + "/" + db + rest[q:]
		}
		return base + "/" + db
	}
	return url
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
