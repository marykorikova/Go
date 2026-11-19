package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNew_InvalidURL(t *testing.T) {
	cfg := Config{
		URL:             "postgres://invalid:password@localhost:abc/test",
		MaxConns:        10,
		MinConns:        1,
		ConnectTimeout:  time.Second,
		MaxConnLifetime: time.Hour,
	}

	pool, err := New(context.Background(), cfg)

	if err == nil {
		t.Fatal("New() expected error, got nil")
	}

	if pool != nil {
		t.Errorf("New() pool = %v, want nil", pool)
	}

	if !strings.Contains(err.Error(), "parse database url") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNew_DatabaseUnavailable(t *testing.T) {
	cfg := Config{
		URL:             "postgres://test:test@127.0.0.1:1/test?sslmode=disable",
		MaxConns:        5,
		MinConns:        0,
		ConnectTimeout:  time.Second,
		MaxConnLifetime: time.Hour,
	}

	pool, err := New(context.Background(), cfg)

	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("New() expected error, got nil")
	}

	if pool != nil {
		pool.Close()
		t.Errorf("New() pool = %v, want nil", pool)
	}

	if !strings.Contains(err.Error(), "ping database") {
		t.Errorf("unexpected error: %v", err)
	}
}
