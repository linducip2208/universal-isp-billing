package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/universal-isp/platform/internal/connectors/lab"
	"github.com/universal-isp/platform/internal/connectors/mikrotik"
)

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "server":
		fmt.Println("use: go run ./cmd/server (see deployments/)")
	case "migrate":
		fmt.Println("apply backend/migrations/*.sql with psql or golang-migrate (see DEVELOPMENT.md)")
	case "seed":
		fmt.Println("apply backend/migrations/002_seed.sql")
	case "worker", "scheduler":
		fmt.Println("starting stub worker loop (Ctrl+C to stop)")
		for {
			time.Sleep(5 * time.Second)
			fmt.Println("tick", time.Now().Format(time.RFC3339))
		}
	case "health":
		fmt.Println(`{"status":"ok"}`)
	case "device":
		// ispctl device test --host X --user Y --pass Z
		host, user, pass := env("MT_HOST", ""), env("MT_USER", "admin"), env("MT_PASS", "")
		if host == "" {
			fmt.Fprintln(os.Stderr, "set MT_HOST/MT_USER/MT_PASS")
			os.Exit(2)
		}
		c := mikrotik.New(mikrotik.Config{Host: host, Username: user, Password: pass})
		res, err := lab.TestAndDiscover(context.Background(), c)
		fmt.Printf("%+v err=%v\n", res, err)
	default:
		fmt.Println("ispctl: server|migrate|seed|worker|scheduler|radius|device|connector|billing|health")
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
