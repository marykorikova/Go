//go:build integration

package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNew_Success(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	cfg := Config{
		URL:             databaseURL,
		MaxConns:        5,
		MinConns:        1,
		ConnectTimeout:  5 * time.Second,
		MaxConnLifetime: time.Hour,
	}

	pool, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping() error: %v", err)
	}

	if pool.Config().MaxConns != cfg.MaxConns {
		t.Errorf("MaxConns = %d, want %d",
			pool.Config().MaxConns, cfg.MaxConns)
	}
}
