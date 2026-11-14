// gantry-cloud is a small simulated IaaS API for the gantry platform.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/georgitinchev/gantry/cloud/internal/api"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	token := os.Getenv("API_TOKEN")
	if len(token) < 16 {
		log.Error("API_TOKEN must be set (16+ characters)")
		os.Exit(1)
	}
	maxInstances, _ := strconv.Atoi(env("MAX_INSTANCES", "20"))
	delay, err := time.ParseDuration(env("PROVISION_DELAY", "6s"))
	if err != nil {
		log.Error("bad PROVISION_DELAY", "err", err)
		os.Exit(1)
	}
	pool, err := netip.ParsePrefix(env("NETWORK_POOL", "10.40.0.0/16"))
	if err != nil {
		log.Error("bad NETWORK_POOL", "err", err)
		os.Exit(1)
	}

	cloud := api.New(api.Config{Pool: pool, MaxInstances: maxInstances, ProvisionDelay: delay})
	srv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           api.NewServer(cloud, token).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	log.Info("gantry-cloud listening", "addr", srv.Addr, "pool", pool.String(), "maxInstances", maxInstances, "provisionDelay", delay.String())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}
