package main

import (
	"context"
	"errors"
	"github.com/marykorikova/Go/internal/service/positionworker"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marykorikova/Go/internal/api"
	"github.com/marykorikova/Go/internal/client/push"
	"github.com/marykorikova/Go/internal/config"
	"github.com/marykorikova/Go/internal/db"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/middleware"
	tripcreatehttp "github.com/marykorikova/Go/internal/http/trip/create"
	tripfinishhttp "github.com/marykorikova/Go/internal/http/trip/finish"
	tripgethttp "github.com/marykorikova/Go/internal/http/trip/get"
	positioncreatehttp "github.com/marykorikova/Go/internal/http/trip/position/create"
	positionlisthttp "github.com/marykorikova/Go/internal/http/trip/position/list"
	"github.com/marykorikova/Go/internal/logger"
	triprepo "github.com/marykorikova/Go/internal/repository/trip"
	"github.com/marykorikova/Go/internal/service/positionscanner"
	"github.com/marykorikova/Go/internal/telemetry"
	"github.com/marykorikova/Go/internal/txmanager"
	tripcreateuc "github.com/marykorikova/Go/internal/usecase/trip/create"
	tripfinishuc "github.com/marykorikova/Go/internal/usecase/trip/finish"
	tripgetuc "github.com/marykorikova/Go/internal/usecase/trip/get"
	positioncreateuc "github.com/marykorikova/Go/internal/usecase/trip/position/create"
	positionlistuc "github.com/marykorikova/Go/internal/usecase/trip/position/list"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	shutdownTracing, err := telemetry.InitTracing(context.Background())
	if err != nil {
		log.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	shutdownMetrics, err := telemetry.InitMetrics(context.Background())
	if err != nil {
		log.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	positionQueueMetrics, err := telemetry.NewPositionQueueMetrics()
	if err != nil {
		log.Error("failed to initialize position queue metrics", "error", err)
		os.Exit(1)
	}

	positionScanMetrics, err := telemetry.NewPositionScanMetrics()
	if err != nil {
		log.Error("failed to initialize position scan metrics", "error", err)
		os.Exit(1)
	}

	positionWorkerMetrics, err := telemetry.NewPositionWorkerMetrics()
	if err != nil {
		log.Error("failed to initialize position worker metrics", "error", err)
		os.Exit(1)
	}

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

	newRepo := triprepo.New(pool)
	createUC := tripcreateuc.New(newRepo, txm)
	createHandler := tripcreatehttp.New(createUC)
	getUC := tripgetuc.New(newRepo)
	getHandler := tripgethttp.New(getUC)
	finishUC := tripfinishuc.New(newRepo, txm)
	finishHandler := tripfinishhttp.New(finishUC)
	positionQueue := positionworker.NewQueue()
	queueSizeRegistration, err := telemetry.RegisterQueueSize(positionQueue.Size)
	if err != nil {
		log.Error("failed to register position queue size metric", "error", err)
		os.Exit(1)
	}
	defer queueSizeRegistration.Unregister()
	positionSavedMetrics, err := telemetry.NewPositionSavedMetrics()
	if err != nil {
		log.Error("failed to create position saved metric", "error", err)
		os.Exit(1)
	}
	positionCreateUC := positioncreateuc.New(newRepo, txm, positionQueue)
	positionCreateUC.SetMetrics(positionSavedMetrics)
	positionCreateHandler := positioncreatehttp.New(positionCreateUC)
	positionListUC := positionlistuc.New(newRepo)
	positionListHandler := positionlisthttp.New(positionListUC)

	pushClient := push.New(cfg.Push.HTTPURL, cfg.Push.RequestTimeout)
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	workerPool := positionworker.NewPool(
		positionQueue,
		positionworker.NewPushHandler(pushClient),
		positionWorkerMetrics,
	)
	activeWorkersRegistration, err := telemetry.RegisterActiveWorkers(workerPool.Active)
	if err != nil {
		log.Error("failed to register active workers metric", "error", err)
		os.Exit(1)
	}
	defer activeWorkersRegistration.Unregister()

	workerPool.Start(workerCtx)
	defer stopWorkers()

	scannerCtx, stopScanner := context.WithCancel(context.Background())
	scannerDone := make(chan struct{})
	scanner := positionscanner.New(
		positionscanner.NewDatabaseScan(newRepo, positionQueue, cfg.PositionStaleAfter, log, positionQueueMetrics),
		cfg.PositionScanInterval,
		positionScanMetrics,
	)

	go func() {
		defer close(scannerDone)
		scanner.Run(scannerCtx)
	}()

	handler := api.NewHandlerWithUsecases(
		pool,
		cfg.Database.QueryTimeout,
		createHandler,
		getHandler,
		finishHandler,
		positionCreateHandler,
		positionListHandler,
	)

	r := chi.NewRouter()
	r.Use(middleware.Logging(log))

	apiGen.HandlerWithOptions(handler, apiGen.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"https://tripgo.example/problems/invalid-request","title":"invalid_request","status":400,"detail":"invalid path parameters","code":"invalid_request"}`))
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           otelhttp.NewHandler(r, "trip-service"),
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

	stopScanner()
	select {
	case <-scannerDone:
		log.Info("position scanner stopped")
	case <-shutdownCtx.Done():
		log.Warn("position scanner shutdown timed out")
	}

	positionQueue.Close()

	workersDone := make(chan struct{})
	go func() {
		workerPool.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		log.Info("position workers stopped")
	case <-shutdownCtx.Done():
		log.Warn("position workers shutdown timed out")
		stopWorkers()
	}

	if err := shutdownTracing(shutdownCtx); err != nil {
		log.Error("failed to shutdown tracing", "error", err)
	}
	if err := shutdownMetrics(shutdownCtx); err != nil {
		log.Error("failed to shutdown metrics", "error", err)
	}
}
