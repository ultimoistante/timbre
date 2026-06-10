// Command server is the Timbre backend entrypoint.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ultimoistante/timbre/internal/api"
	"github.com/ultimoistante/timbre/internal/config"
	"github.com/ultimoistante/timbre/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	gdb, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	srv := api.New(cfg, gdb)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	go func() {
		log.Printf("timbre listening on %s (data dir: %s, db: %s)", addr, cfg.DataDir, cfg.DBDriver)
		if err := srv.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Handler().Shutdown(ctx); err != nil {
		// Active streaming connections may outlive the timeout; force-close them.
		_ = srv.Handler().Close()
	}

	if sqlDB, err := gdb.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
