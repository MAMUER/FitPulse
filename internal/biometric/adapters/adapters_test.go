package adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/biometric/config"
	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/logger"
)

func TestNewPlatformAdapter(t *testing.T) {
	adapter := NewPlatformAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", map[domain.MetricType]bool{
		domain.MetricHeartRate: true,
	}, 100*time.Millisecond)

	require.NotNil(t, adapter)
	assert.Equal(t, "test", adapter.DeviceType())
	assert.True(t, adapter.Supports("heart_rate"))
	assert.False(t, adapter.Supports("spo2"))
}

func TestPlatformAdapter_Fetch(t *testing.T) {
	adapter := NewPlatformAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", map[domain.MetricType]bool{
		domain.MetricHeartRate: true,
	}, 0)

	samples, err := adapter.Fetch(context.Background(), "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Empty(t, samples)
}

func TestPlatformAdapter_Supports(t *testing.T) {
	adapter := NewPlatformAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", map[domain.MetricType]bool{
		domain.MetricHeartRate:   true,
		domain.MetricSpO2:        true,
		domain.MetricTemperature: false,
	}, 100*time.Millisecond)

	assert.True(t, adapter.Supports("heart_rate"))
	assert.True(t, adapter.Supports("spo2"))
	assert.False(t, adapter.Supports("temperature"))
	assert.False(t, adapter.Supports("unknown_metric"))
}

func TestPlatformAdapter_DeviceType(t *testing.T) {
	adapter := NewPlatformAdapter("fitbit", "https://api.fitbit.com", "key", "user-1", "device-1", nil, 100*time.Millisecond)
	assert.Equal(t, "fitbit", adapter.DeviceType())
}

func TestPlatformAdapter_HealthCheck(t *testing.T) {
	adapter := NewPlatformAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", nil, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := adapter.HealthCheck(ctx)
	assert.Error(t, err)
}

func TestPlatformAdapter_HealthCheck_InvalidURL(t *testing.T) {
	adapter := NewPlatformAdapter("test", "http://", "api-key", "user-1", "device-1", nil, 100*time.Millisecond)

	ctx := context.Background()
	err := adapter.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execute health check")
}

func TestPlatformAdapter_HealthCheck_InvalidURLCreateError(t *testing.T) {
	adapter := NewPlatformAdapter("test", "http://[invalid", "api-key", "user-1", "device-1", nil, 100*time.Millisecond)

	ctx := context.Background()
	err := adapter.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create health check request")
}

func TestPlatformAdapter_HealthCheck_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewPlatformAdapter("test", server.URL, "api-key", "user-1", "device-1", nil, 100*time.Millisecond)

	ctx := context.Background()
	err := adapter.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestPlatformAdapter_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewPlatformAdapter("test", server.URL, "api-key", "user-1", "device-1", nil, 100*time.Millisecond)

	ctx := context.Background()
	err := adapter.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestNewAdapterFactory(t *testing.T) {
	factory := NewAdapterFactory(map[string]string{
		"fitbit": "fitbit-api-key",
		"garmin": "garmin-api-key",
	})

	require.NotNil(t, factory)
	assert.NotEmpty(t, factory.apiKeys)
	assert.NotNil(t, factory.config)
	assert.NotNil(t, factory.caps)
}

func TestAdapterFactory_CreateAdapter(t *testing.T) {
	factory := NewAdapterFactory(map[string]string{
		"fitbit": "fitbit-api-key",
		"garmin": "garmin-api-key",
	})

	t.Run("creates fitbit adapter", func(t *testing.T) {
		adapter, err := factory.CreateAdapter("fitbit", "user-1", "device-1")
		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.Equal(t, "fitbit", adapter.DeviceType())
		assert.True(t, adapter.Supports("heart_rate"))
	})

	t.Run("creates garmin adapter", func(t *testing.T) {
		adapter, err := factory.CreateAdapter("garmin", "user-1", "device-1")
		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.Equal(t, "garmin", adapter.DeviceType())
		assert.True(t, adapter.Supports("temperature"))
	})

	t.Run("creates withings adapter", func(t *testing.T) {
		factoryWithings := NewAdapterFactory(map[string]string{
			"withings": "withings-api-key",
		})
		adapter, err := factoryWithings.CreateAdapter("withings", "user-1", "device-1")
		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.Equal(t, "withings", adapter.DeviceType())
	})

	t.Run("creates rook adapter with empty caps", func(t *testing.T) {
		factoryRook := NewAdapterFactory(map[string]string{
			"rook": "rook-api-key",
		})
		adapter, err := factoryRook.CreateAdapter("rook", "user-1", "device-1")
		require.NoError(t, err)
		require.NotNil(t, adapter)
		assert.Equal(t, "rook", adapter.DeviceType())
		assert.False(t, adapter.Supports("heart_rate"))
	})

	t.Run("returns error for unknown device type", func(t *testing.T) {
		adapter, err := factory.CreateAdapter("unknown", "user-1", "device-1")
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "no configuration for device type")
	})

	t.Run("returns error when API key missing", func(t *testing.T) {
		factoryNoKey := NewAdapterFactory(map[string]string{})
		adapter, err := factoryNoKey.CreateAdapter("fitbit", "user-1", "device-1")
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "no API key configured")
	})

	t.Run("returns error when device type disabled", func(t *testing.T) {
		factoryDisabled := NewAdapterFactory(map[string]string{
			"fitbit": "fitbit-api-key",
		})
		adapterConfig := factoryDisabled.config
		adapterConfig.Vendors["fitbit"] = config.VendorConfig{
			BaseURL:    adapterConfig.Vendors["fitbit"].BaseURL,
			APIKey:     adapterConfig.Vendors["fitbit"].APIKey,
			DeviceType: adapterConfig.Vendors["fitbit"].DeviceType,
			RateLimit:  adapterConfig.Vendors["fitbit"].RateLimit,
			Timeout:    adapterConfig.Vendors["fitbit"].Timeout,
			Enabled:    false,
		}
		factoryDisabled.config = adapterConfig
		adapter, err := factoryDisabled.CreateAdapter("fitbit", "user-1", "device-1")
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "device type disabled")
	})
}

func TestAdapterFactory_Supports(t *testing.T) {
	factory := NewAdapterFactory(map[string]string{
		"fitbit": "fitbit-api-key",
		"garmin": "garmin-api-key",
	})

	assert.True(t, factory.Supports("fitbit", "heart_rate"))
	assert.True(t, factory.Supports("garmin", "temperature"))
	assert.False(t, factory.Supports("fitbit", "temperature"))
	assert.False(t, factory.Supports("unknown", "heart_rate"))
}

func TestBaseAdapter_WaitForRateLimit(t *testing.T) {
	adapter := newBaseAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", 300*time.Millisecond)

	start := time.Now()
	adapter.waitForRateLimit()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 10*time.Millisecond)

	start = time.Now()
	adapter.waitForRateLimit()
	elapsed = time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond)
}

func TestBaseAdapter_WaitForRateLimit_Concurrent(t *testing.T) {
	adapter := newBaseAdapter("test", "https://api.example.com", "api-key", "user-1", "device-1", 300*time.Millisecond)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			adapter.waitForRateLimit()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent rate limit calls")
		}
	}
}

func TestNewLoggerAdapter(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		samples:    []domain.BiometricSample{},
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	require.NotNil(t, loggerAdapter)
	assert.Equal(t, "test", loggerAdapter.DeviceType())
}

func TestLoggerAdapter_Fetch(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		samples: []domain.BiometricSample{
			{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm"},
		},
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	samples, err := loggerAdapter.Fetch(context.Background(), "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
	assert.Equal(t, "test", loggerAdapter.DeviceType())
}

func TestLoggerAdapter_Fetch_Error(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		err:        errors.New("fetch failed"),
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	samples, err := loggerAdapter.Fetch(context.Background(), "user-1", []string{"heart_rate"})
	assert.Error(t, err)
	assert.Nil(t, samples)
	assert.Equal(t, "fetch failed", err.Error())
}

func TestLoggerAdapter_Supports(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	assert.True(t, loggerAdapter.Supports("heart_rate"))
	assert.False(t, loggerAdapter.Supports("spo2"))
}

func TestLoggerAdapter_DeviceType(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{},
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	assert.Equal(t, "test", loggerAdapter.DeviceType())
}

func TestLoggerAdapter_HealthCheck(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	err := loggerAdapter.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestLoggerAdapter_HealthCheck_Error(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		err:        errors.New("health check failed"),
	}

	loggerAdapter := NewLoggerAdapter(source, logger.New("test"))
	err := loggerAdapter.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "health check failed", err.Error())
}

func TestNewMetricsAdapter(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		samples:    []domain.BiometricSample{},
	}

	fetchCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_fetch_total",
			Help: "Test fetch counter",
		},
		[]string{"source", "status"},
	)
	fetchDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_fetch_duration_seconds",
			Help:    "Test fetch duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source"},
	)

	adapter := NewMetricsAdapter(source, fetchCounter, fetchDuration)
	require.NotNil(t, adapter)
	assert.Equal(t, "test", adapter.DeviceType())
}

func TestMetricsAdapter_Fetch(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		samples: []domain.BiometricSample{
			{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm"},
		},
	}

	fetchCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_fetch_total",
			Help: "Test fetch counter",
		},
		[]string{"source", "status"},
	)
	fetchDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_fetch_duration_seconds",
			Help:    "Test fetch duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source"},
	)

	adapter := NewMetricsAdapter(source, fetchCounter, fetchDuration)
	samples, err := adapter.Fetch(context.Background(), "user-1", []string{"heart_rate"})
	require.NoError(t, err)
	assert.Len(t, samples, 1)
	assert.Equal(t, "test", adapter.DeviceType())
}

func TestMetricsAdapter_Fetch_Error(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
		err:        errors.New("fetch failed"),
	}

	fetchCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_fetch_total",
			Help: "Test fetch counter",
		},
		[]string{"source", "status"},
	)
	fetchDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_fetch_duration_seconds",
			Help:    "Test fetch duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source"},
	)

	adapter := NewMetricsAdapter(source, fetchCounter, fetchDuration)
	samples, err := adapter.Fetch(context.Background(), "user-1", []string{"heart_rate"})
	assert.Error(t, err)
	assert.Nil(t, samples)
}

func TestMetricsAdapter_Supports(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
	}

	adapter := NewMetricsAdapter(source, nil, nil)
	assert.True(t, adapter.Supports("heart_rate"))
	assert.False(t, adapter.Supports("spo2"))
}

func TestMetricsAdapter_DeviceType(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
		supports:   map[domain.MetricType]bool{},
	}

	adapter := NewMetricsAdapter(source, nil, nil)
	assert.Equal(t, "test", adapter.DeviceType())
}

func TestMetricsAdapter_HealthCheck(t *testing.T) {
	source := &stubSource{
		deviceType: "test",
	}

	adapter := NewMetricsAdapter(source, nil, nil)
	err := adapter.HealthCheck(context.Background())
	assert.NoError(t, err)
}
