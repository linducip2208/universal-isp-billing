package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/universal-isp/platform/internal/config"
	"github.com/universal-isp/platform/internal/connectors/all"
	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/health"
	"github.com/universal-isp/platform/internal/httpapi"
	"github.com/universal-isp/platform/internal/logger"
	"github.com/universal-isp/platform/internal/store"
)

// Connector registration lives in internal/connectors/all (shared by the
// API server and ispctl so both always agree).
func registerAll() { all.RegisterAll() }

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel, os.Stdout)
	registerAll()
	var st *store.Store
	if cfg.DatabaseURL == "" {
		log.Warn("DATABASE_URL empty — API runs without database (all data routes answer 503)")
	} else {
		db, err := database.Open(cfg.DatabaseURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "database:", err)
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = database.Ping(ctx, db)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "database unreachable:", err)
			os.Exit(1)
		}
		st = store.New(db)
	}
	srv := httpapi.New(log, cfg.JWTSecret, &health.Checker{RedisAddr: cfg.RedisAddr}).WithStore(st)
	if os.Getenv("ISP_DEMO_LOGIN") == "1" {
		log.Warn("ISP_DEMO_LOGIN=1 — demo admin/secret login ENABLED (development only)")
		srv.WithDemoLogin(true)
	}
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		srv.WithCORS(strings.Split(origins, ","))
	}
	httpSrv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Info("http listening", slog.Int("port", cfg.HTTPPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("listen", slog.String("err", err.Error()))
		}
	}()
	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shut)
}
