package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/universal-isp/platform/internal/config"
	"github.com/universal-isp/platform/internal/connectors/cloud"
	"github.com/universal-isp/platform/internal/connectors/generic"
	"github.com/universal-isp/platform/internal/connectors/mikrotik"
	"github.com/universal-isp/platform/internal/connectors/olt"
	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/routers"
	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/health"
	"github.com/universal-isp/platform/internal/httpapi"
	"github.com/universal-isp/platform/internal/logger"
)

func registerAll() {
	registry.Register(registry.Entry{Vendor: "MikroTik", ProductFamily: "RouterOS", ConnectionType: sdk.ConnCLI, Status: sdk.StateVerified, DocURL: "docs/connectors/mikrotik.md", Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return mikrotik.New(mikrotik.Config{Host: cfg["host"], Username: cfg["username"], Password: cfg["password"]}), nil
	}})
	registry.Register(registry.Entry{Vendor: "MikroTik", ProductFamily: "RouterOS-REST", ConnectionType: sdk.ConnREST, Status: sdk.StateImplemented, DocURL: "docs/connectors/mikrotik.md", Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return mikrotik.New(mikrotik.Config{Host: cfg["host"], Username: cfg["username"], Password: cfg["password"], UseREST: true, UseTLS: true}), nil
	}})
	registry.Register(registry.Entry{Vendor: "Ruijie", ProductFamily: "Cloud", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/ruijie-cloud.md", Factory: cloud.NewRuijieCloud})
	registry.Register(registry.Entry{Vendor: "Ruijie", ProductFamily: "Reyee", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/reyee.md", Factory: cloud.NewReyee})
	registry.Register(registry.Entry{Vendor: "Ubiquiti", ProductFamily: "UniFi", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/unifi.md", Factory: cloud.NewUniFi})
	registry.Register(registry.Entry{Vendor: "TP-Link", ProductFamily: "Omada", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/omada.md", Factory: cloud.NewOmada})
	registry.Register(registry.Entry{Vendor: "Cisco", ProductFamily: "Meraki", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/meraki.md", Factory: cloud.NewMeraki})
	registry.Register(registry.Entry{Vendor: "Aruba", ProductFamily: "Central", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/aruba-central.md", Factory: cloud.NewArubaCentral})
	registry.Register(registry.Entry{Vendor: "Juniper", ProductFamily: "Mist", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/mist.md", Factory: cloud.NewMist})
	registry.Register(registry.Entry{Vendor: "Cambium", ProductFamily: "cnMaestro", ConnectionType: sdk.ConnCloudAPI, Status: sdk.StateRequiresVendorAcc, DocURL: "docs/connectors/cnmaestro.md", Factory: cloud.NewCnMaestro})
	for name, f := range routers.Factories {
		vendor, family := "vendor", name
		_ = vendor
		_ = family
		_ = f
	}
	// Router direct families
	routerFams := []struct {
		v, f string
		ct   sdk.ConnectionType
	}{
		{"Cisco", "IOS/XE", sdk.ConnSSH}, {"Juniper", "Junos", sdk.ConnNETCONF},
		{"Huawei", "VRP", sdk.ConnSSH}, {"ZTE", "ZXAN", sdk.ConnSSH},
		{"Nokia", "SR OS", sdk.ConnNETCONF}, {"VyOS", "VyOS", sdk.ConnRESTCONF},
		{"Ubiquiti", "EdgeRouter", sdk.ConnSSH}, {"Fortinet", "FortiOS", sdk.ConnREST},
		{"Aruba", "CX", sdk.ConnREST}, {"Ruijie", "Router", sdk.ConnSSH},
		{"Cisco", "SmallBusiness", sdk.ConnREST}, {"D-Link", "Smart", sdk.ConnSNMP},
	}
	for _, r := range routerFams {
		v, f, ct := r.v, r.f, r.ct
		_ = routers.Factories
		registry.Register(registry.Entry{Vendor: v, ProductFamily: f, ConnectionType: ct, Status: sdk.StateModelDependent, DocURL: "docs/connectors/generic.md", Factory: routers.Factories["cisco-ios"]})
		_ = v
		_ = f
	}
	oltFams := [][2]string{{"Huawei", "MA5600T"}, {"ZTE", "C320"}, {"Nokia", "7360"}, {"FiberHome", "AN5116"}, {"BDCOM", "P3310"}, {"VSOL", "V1600"}, {"C-Data", "FD1608"}, {"Dasan", "V8240"}, {"Raisecom", "ISC"},
		{"Zyxel", "OLT1408"}, {"Calix", "E7"}, {"ADTRAN", "TA5000"}, {"DZS", "Velocity"}}
	for _, o := range oltFams {
		v, f := o[0], o[1]
		registry.Register(registry.Entry{Vendor: v, ProductFamily: f, ConnectionType: sdk.ConnSNMP, Status: sdk.StateModelDependent, DocURL: "docs/connectors/olt.md", Factory: olt.Mk(v, f, sdk.ConnSNMP)})
	}
	// Generics
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "RADIUS", ConnectionType: sdk.ConnRADIUS, Status: sdk.StateImplemented, Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return generic.NewRADIUS(cfg["nas_addr"], cfg["secret"]), nil
	}})
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "SNMP", ConnectionType: sdk.ConnSNMP, Status: sdk.StateImplemented, Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return generic.NewSNMP(cfg["target"], cfg["community"]), nil
	}})
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "REST", ConnectionType: sdk.ConnGenericHTTP, Status: sdk.StateImplemented, Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return generic.NewREST(generic.RESTConfig{BaseURL: cfg["base_url"], Token: cfg["token"]}), nil
	}})
}

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel, os.Stdout)
	registerAll()
	h := &health.Checker{RedisAddr: cfg.RedisAddr}
	srv := httpapi.New(log, cfg.JWTSecret, h)
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
