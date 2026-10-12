package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("returns config with provided ports", func(t *testing.T) {
		cfg := DefaultConfig("8080", "9090")
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "9090", cfg.MetricsPort)
	})

	t.Run("returns default timeouts", func(t *testing.T) {
		cfg := DefaultConfig("8080", "9090")
		assert.Equal(t, 15*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 10*time.Second, cfg.ShutdownTimeout)
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("empty ports are accepted", func(t *testing.T) {
		cfg := DefaultConfig("", "")
		assert.Equal(t, "", cfg.Port)
		assert.Equal(t, "", cfg.MetricsPort)
	})

	t.Run("custom timeouts are preserved", func(t *testing.T) {
		cfg := Config{
			Port:            "8080",
			MetricsPort:     "9090",
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     15 * time.Second,
			ShutdownTimeout: 3 * time.Second,
		}
		assert.Equal(t, 5*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 15*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 3*time.Second, cfg.ShutdownTimeout)
	})
}

const (
	serveHelperEnv        = "FITPULSE_SERVE_HELPER"
	serveHelperMainPort   = "FITPULSE_SERVE_HELPER_MAIN_PORT"
	serveHelperMetricsPrt = "FITPULSE_SERVE_HELPER_METRICS_PORT"
)

func silentLogger() *logger.Logger {
	return &logger.Logger{Logger: zap.NewNop()}
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	require.NoError(t, ln.Close())
	return port
}

func serveConfig(mainPort, metricsPort string) Config {
	return Config{
		Port:            mainPort,
		MetricsPort:     metricsPort,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     5 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

func healthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func metricsMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("metrics"))
	})
	return mux
}

func getBody(t *testing.T, port, path string) (int, string, error) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:" + port + path)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(body), nil
}

func waitForEndpoint(t *testing.T, port, path string) {
	t.Helper()
	require.Eventually(t, func() bool {
		status, _, err := getBody(t, port, path)
		return err == nil && status == http.StatusOK
	}, 10*time.Second, 25*time.Millisecond)
}

func fetchBody(t *testing.T, port, path string) (int, string) {
	t.Helper()
	status, body, err := getBody(t, port, path)
	require.NoError(t, err)
	return status, body
}

func serveInBackground(cfg Config) chan struct{} {
	done := make(chan struct{})
	go func() {
		Serve(silentLogger(), cfg, metricsMux(), healthMux(), healthMux())
		close(done)
	}()
	return done
}

func TestServeServesMainAndMetricsEndpoints(t *testing.T) {
	mainPort := freePort(t)
	metricsPort := freePort(t)
	done := serveInBackground(serveConfig(mainPort, metricsPort))

	waitForEndpoint(t, mainPort, "/health")

	status, body := fetchBody(t, mainPort, "/health")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ok", body)

	status, body = fetchBody(t, metricsPort, "/metrics")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "metrics", body)

	status, _ = fetchBody(t, mainPort, "/metrics")
	assert.Equal(t, http.StatusNotFound, status)

	select {
	case <-done:
		t.Fatal("Serve returned before a termination signal was received")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestServeHelperProcess(t *testing.T) {
	if os.Getenv(serveHelperEnv) != "1" {
		t.Skip("only runs as a child process spawned by TestServeShutsDownOnTerminationSignal")
	}

	mainPort := os.Getenv(serveHelperMainPort)
	metricsPort := os.Getenv(serveHelperMetricsPrt)
	require.NotEmpty(t, mainPort)
	require.NotEmpty(t, metricsPort)

	done := serveInBackground(serveConfig(mainPort, metricsPort))

	waitForEndpoint(t, mainPort, "/health")
	fmt.Println("serve helper: servers are up")

	select {
	case <-done:
		fmt.Println("serve helper: Serve returned")
	case <-time.After(30 * time.Second):
		fmt.Println("serve helper: Serve did not return in time")
		os.Exit(2)
	}
}

func TestServeShutsDownOnTerminationSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not deliver termination signals to child processes")
	}

	mainPort := freePort(t)
	metricsPort := freePort(t)

	cmd := exec.Command(os.Args[0], "-test.run=^TestServeHelperProcess$", "-test.timeout=60s")
	cmd.Env = append(os.Environ(),
		serveHelperEnv+"=1",
		serveHelperMainPort+"="+mainPort,
		serveHelperMetricsPrt+"="+metricsPort,
	)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	require.NoError(t, cmd.Start())

	waitForEndpoint(t, mainPort, "/health")

	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	select {
	case err := <-waitErr:
		require.NoError(t, err)
		assert.Contains(t, cmd.Stdout.(*bytes.Buffer).String(), "serve helper: Serve returned")
	case <-time.After(45 * time.Second):
		_ = cmd.Process.Kill()
		<-waitErr
		t.Fatalf("Serve did not shut down after a termination signal\nSTDOUT:\n%s\nSTDERR:\n%s",
			cmd.Stdout.(*bytes.Buffer).String(), cmd.Stderr.(*bytes.Buffer).String())
	}
}
