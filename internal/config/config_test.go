package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("PUSH_HTTP_URL", "http://localhost:8090")
	t.Setenv("PUSH_REQUEST_TIMEOUT", "")

	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONNS", "")
	t.Setenv("DATABASE_MIN_CONNS", "")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}

	if cfg.Database.URL != "postgres://localhost:5432/test" {
		t.Errorf("Database.URL = %q, unexpected value", cfg.Database.URL)
	}

	if cfg.Database.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want 10", cfg.Database.MaxConns)
	}

	if cfg.Database.MinConns != 1 {
		t.Errorf("MinConns = %d, want 1", cfg.Database.MinConns)
	}

	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Errorf("ConnectTimeout = %v, want 5s", cfg.Database.ConnectTimeout)
	}

	if cfg.Database.QueryTimeout != 5*time.Second {
		t.Errorf("QueryTimeout = %v, want 5s", cfg.Database.QueryTimeout)
	}

	if cfg.Database.MaxConnLifetime != 30*time.Minute {
		t.Errorf("MaxConnLifetime = %v, want 30m", cfg.Database.MaxConnLifetime)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}

	if cfg != nil {
		t.Errorf("Load() config = %v, want nil", cfg)
	}

	want := "DATABASE_URL is required"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoad_InvalidMaxConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("PUSH_HTTP_URL", "http://localhost:8090")
	t.Setenv("PUSH_REQUEST_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONNS", "0")
	t.Setenv("DATABASE_MIN_CONNS", "1")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}

	if cfg != nil {
		t.Errorf("Load() config = %v, want nil", cfg)
	}

	want := "DATABASE_MAX_CONNS must be > 0"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoad_InvalidMinConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("PUSH_HTTP_URL", "http://localhost:8090")
	t.Setenv("PUSH_REQUEST_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONNS", "10")
	t.Setenv("DATABASE_MIN_CONNS", "-1")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}

	if cfg != nil {
		t.Errorf("Load() config = %v, want nil", cfg)
	}

	want := "DATABASE_MIN_CONNS must be >= 0"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoad_MinConnsGreaterThanMaxConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("PUSH_HTTP_URL", "http://localhost:8090")
	t.Setenv("PUSH_REQUEST_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONNS", "10")
	t.Setenv("DATABASE_MIN_CONNS", "15")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")

	cfg, err := Load()

	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}

	if cfg != nil {
		t.Errorf("Load() config = %v, want nil", cfg)
	}

	want := "DATABASE_MIN_CONNS must be <= DATABASE_MAX_CONNS"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoad_PushConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
	t.Setenv("PUSH_HTTP_URL", "http://localhost:23809")
	t.Setenv("PUSH_REQUEST_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Push.HTTPURL != "http://localhost:23809" {
		t.Errorf("Push.HTTPURL = %q, want http://localhost:23809", cfg.Push.HTTPURL)
	}

	if cfg.Push.RequestTimeout != time.Second {
		t.Errorf("Push.RequestTimeout = %v, want 1s", cfg.Push.RequestTimeout)
	}
}

func TestLoad_PositionScanInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "default", value: "", want: 30 * time.Second},
		{name: "custom", value: "5s", want: 5 * time.Second},
		{name: "zero", value: "0s", wantErr: true},
		{name: "negative", value: "-1s", wantErr: true},
		{name: "invalid", value: "abc", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
			t.Setenv("PUSH_HTTP_URL", "http://localhost:8090")
			t.Setenv("POSITION_CHECK_INTERVAL", tc.value)

			cfg, err := Load()
			if tc.wantErr {
				if err == nil {
					t.Fatal("Load() expected error, got nil")
				}
				if cfg != nil {
					t.Errorf("Load() config = %v, want nil", cfg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.PositionScanInterval != tc.want {
				t.Errorf("PositionScanInterval = %v, want %v",
					cfg.PositionScanInterval, tc.want)
			}
		})
	}
}
