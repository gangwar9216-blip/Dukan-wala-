package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"healthcare-backend/internal/auth"
	"healthcare-backend/internal/config"
	"healthcare-backend/internal/db"
	"healthcare-backend/internal/httpapi"
	"healthcare-backend/internal/store"
)

func main() {
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := db.RunMigrations(migrationCtx, pool); err != nil {
		migrationCancel()
		log.Fatal(err)
	}
	migrationCancel()

	srv := httpapi.Server{
		Store:                 store.Store{DB: pool},
		Auth:                  auth.Manager{Secret: []byte(cfg.JWTSecret)},
		RazorpayKeyID:         cfg.RazorpayKeyID,
		RazorpayKeySecret:     cfg.RazorpayKeySecret,
		RazorpayWebhookSecret: cfg.RazorpayWebhookSecret,
		OTPDevMode:            cfg.OTPDevMode,
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("healthcare backend listening on :%s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
