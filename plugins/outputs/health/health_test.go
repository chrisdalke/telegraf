package health_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/outputs/health"
	"github.com/influxdata/telegraf/testutil"
)

var pki = testutil.NewPKI("../../../testutil/pki")

func TestHealth(t *testing.T) {
	type Options struct {
		Compares []*health.Compares `toml:"compares"`
		Contains []*health.Contains `toml:"contains"`
	}

	now := time.Now()
	tests := []struct {
		name         string
		options      Options
		metrics      []telegraf.Metric
		expectedCode int
	}{
		{
			name:         "healthy on startup",
			expectedCode: 200,
		},
		{
			name: "check passes",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						GT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
			},
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 200,
		},
		{
			name: "check fails",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						LT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
			},
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 503,
		},
		{
			name: "mixed check fails",
			options: Options{
				Compares: []*health.Compares{
					{
						Field: "time_idle",
						LT:    func() *float64 { v := 0.0; return &v }(),
					},
				},
				Contains: []*health.Contains{
					{
						Field: "foo",
					},
				},
			},
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{
						"time_idle": 42,
					},
					now),
			},
			expectedCode: 503,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := health.NewHealth()
			output.ServiceAddress = "tcp://127.0.0.1:0"
			output.Compares = tt.options.Compares
			output.Contains = tt.options.Contains
			output.Log = testutil.Logger{}

			err := output.Init()
			require.NoError(t, err)

			err = output.Connect()
			require.NoError(t, err)

			err = output.Write(tt.metrics)
			require.NoError(t, err)

			resp, err := http.Get(output.Origin())
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tt.expectedCode, resp.StatusCode)

			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = output.Close()
			require.NoError(t, err)
		})
	}
}

func TestInitServiceAddress(t *testing.T) {
	tests := []struct {
		name   string
		plugin *health.Health
		err    bool
		origin string
	}{
		{
			name: "port without scheme is not allowed",
			plugin: &health.Health{
				ServiceAddress: ":8080",
				Log:            testutil.Logger{},
			},
			err: true,
		},
		{
			name: "path without scheme is not allowed",
			plugin: &health.Health{
				ServiceAddress: "/tmp/telegraf",
				Log:            testutil.Logger{},
			},
			err: true,
		},
		{
			name: "tcp with port maps to http",
			plugin: &health.Health{
				ServiceAddress: "tcp://:8080",
				Log:            testutil.Logger{},
			},
		},
		{
			name: "tcp with tlsconf maps to https",
			plugin: &health.Health{
				ServiceAddress: "tcp://:8080",
				ServerConfig:   *pki.TLSServerConfig(),
				Log:            testutil.Logger{},
			},
		},
		{
			name: "tcp4 is allowed",
			plugin: &health.Health{
				ServiceAddress: "tcp4://:8080",
				Log:            testutil.Logger{},
			},
		},
		{
			name: "tcp6 is allowed",
			plugin: &health.Health{
				ServiceAddress: "tcp6://:8080",
				Log:            testutil.Logger{},
			},
		},
		{
			name: "http scheme",
			plugin: &health.Health{
				ServiceAddress: "http://:8080",
				Log:            testutil.Logger{},
			},
		},
		{
			name: "https scheme",
			plugin: &health.Health{
				ServiceAddress: "https://:8080",
				Log:            testutil.Logger{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := health.NewHealth()
			output.ServiceAddress = tt.plugin.ServiceAddress
			output.Log = testutil.Logger{}

			err := output.Init()
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTimeBetweenMetrics(t *testing.T) {
	arbitraryTime := time.Time{}.AddDate(2002, 0, 0)
	tests := []struct {
		name                  string
		maxTimeBetweenMetrics config.Duration
		metrics               []telegraf.Metric
		delay                 time.Duration
		expectedCode          int
	}{
		{
			name:                  "healthy enabled no metrics before timeout",
			maxTimeBetweenMetrics: config.Duration(1 * time.Second),
			metrics:               nil,
			delay:                 0 * time.Second,
			expectedCode:          200,
		},
		{
			name:                  "unhealthy enabled no metrics after timeout",
			maxTimeBetweenMetrics: config.Duration(5 * time.Millisecond),
			metrics:               nil,
			delay:                 5 * time.Millisecond,
			expectedCode:          503,
		},
		{
			name:                  "healthy when disabled and old metric",
			maxTimeBetweenMetrics: config.Duration(0),
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]any{
						"time_idle": 42,
					},
					arbitraryTime),
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]any{
						"time_idle": 64,
					},
					arbitraryTime),
			},
			delay:        10 * time.Millisecond,
			expectedCode: 200,
		},
		{
			name:                  "healthy when enabled and recent metric",
			maxTimeBetweenMetrics: config.Duration(5 * time.Second),
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]any{
						"time_idle": 42,
					},
					arbitraryTime),
			},
			delay:        0 * time.Second,
			expectedCode: 200,
		},
		{
			name:                  "unhealthy when enabled and old metric",
			maxTimeBetweenMetrics: config.Duration(5 * time.Millisecond),
			metrics: []telegraf.Metric{
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]any{
						"time_idle": 42,
					},
					arbitraryTime),
				testutil.MustMetric(
					"cpu",
					map[string]string{},
					map[string]any{
						"time_idle": 64,
					},
					arbitraryTime),
			},
			delay:        10 * time.Millisecond,
			expectedCode: 503,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dut := health.NewHealth()
			dut.ServiceAddress = "tcp://127.0.0.1:0"
			dut.Log = testutil.Logger{}
			dut.MaxTimeBetweenMetrics = tt.maxTimeBetweenMetrics

			err := dut.Init()
			require.NoError(t, err)

			err = dut.Connect()
			require.NoError(t, err)

			err = dut.Write(tt.metrics)
			require.NoError(t, err)

			time.Sleep(tt.delay)
			resp, err := http.Get(dut.Origin())
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tt.expectedCode, resp.StatusCode)

			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)

			err = dut.Close()
			require.NoError(t, err)
		})
	}
}
