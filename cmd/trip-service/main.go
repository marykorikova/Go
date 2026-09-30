package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/api"
	"github.com/marykorikova/Go/internal/config"
	"github.com/marykorikova/Go/internal/db"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/logger"
	"github.com/marykorikova/Go/internal/repository"
	"github.com/marykorikova/Go/internal/service"
	"github.com/marykorikova/Go/internal/txmanager"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	pool, err := db.New(ctx, db.Config{
		URL:             cfg.Database.URL,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		ConnectTimeout:  cfg.Database.ConnectTimeout,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
	})
	cancel()
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	txm := txmanager.New(pool)
	repo := repository.NewTripRepository(pool)
	svc := service.NewTripService(repo, txm)
	handler := api.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/api/v1/trips", func(w http.ResponseWriter, r *http.Request) {
		handler.CreateTrip(w, r, apiGen.CreateTripParams{})
	})
	
	r.Get("/api/v1/trips/{tripId}", func(w http.ResponseWriter, r *http.Request) {
		tripId := chi.URLParam(r, "tripId")
		// Преобразуем строку в UUID (можно добавить обработку ошибки)
		uuid, _ := uuid.Parse(tripId)
		handler.GetTrip(w, r, uuid)
	})

	r.Post("/api/v1/trips/{tripId}/finish", func(w http.ResponseWriter, r *http.Request) {
		tripId := chi.URLParam(r, "tripId")
		uuid, _ := uuid.Parse(tripId)
		handler.FinishTrip(w, r, uuid)
	})

	r.Get("/health", handler.Health)
	r.Get("/ready", handler.Ready)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("server started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
	log.Info("server stopped")
}
