package config

import (
	"log/slog"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestIsBucket(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["priming_responses"]
	isBucket, params := metricConfig.IsBucket("ReplyLen")
	assert.True(t, isBucket)
	assert.Equal(t, BucketParams{Start: -1, Width: 50, Count: 0}, params)
}

func TestIsEliminateDimension(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["pcap_stats"]
	isEliminateDimension := metricConfig.IsEliminateDimension("ifname")
	assert.True(t, isEliminateDimension)
}

func TestIsMaxCells(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["second_ld_vs_rcode"]
	isMaxCells, params := metricConfig.IsMaxCells("SecondLD")
	assert.True(t, isMaxCells)
	assert.Equal(t, MaxCellsParams{X: 5}, params)
}

func TestIsFilter(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["qtype"]
	isFilter, allowedValues := metricConfig.IsFilter("Qtype")
	assert.True(t, isFilter)
	slices.Sort(allowedValues)
	assert.Equal(t, []string{"A", "AAAA", "NS"}, allowedValues)
}

func TestIsFilterButItsNot(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["second_ld_vs_rcode"]
	isFilter, allowedValues := metricConfig.IsFilter("SecondLD")
	assert.False(t, isFilter)
	assert.Equal(t, []string{}, allowedValues)
}

func TestGetLogLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, GetLogLevel("debug"))
	assert.Equal(t, slog.LevelInfo, GetLogLevel("info"))
	assert.Equal(t, slog.LevelWarn, GetLogLevel("warn"))
	assert.Equal(t, slog.LevelError, GetLogLevel("error"))
	assert.Equal(t, slog.LevelInfo, GetLogLevel("gibtsnet"))
}

func TestBucketBorders(t *testing.T) {
	params := BucketParams{
		Start: 31,
		Width: 32,
		Count: 100,
	}
	expectedBucketBorders := prometheus.LinearBuckets(float64(params.Start), float64(params.Width), params.Count)
	assert.Equal(t, expectedBucketBorders, params.Buckets())
}

func TestIsInTimeWindow(t *testing.T) {
	promConfig := PrometheusConfig{
		Metrics:    map[string]MetricConfig{},
		Port:       2112,
		Timestamps: true,
		WindowSize: 5,
	}
	assert.True(t, promConfig.IsInTimeWindow(time.Now().Add(-5*time.Minute+time.Second)))
	assert.False(t, promConfig.IsInTimeWindow(time.Now().Add(-5*time.Minute-1*time.Second)))
}

func TestConfig(t *testing.T) {

	config := ParseConfig("./testdata/config.yaml")

	expectedConfig := Config{
		RemoveReadFiles: true,
		DataDir:         DefaultDataDir,
		Interval:        20 * time.Second,
		LogLevel:        slog.LevelWarn,
		Prometheus: PrometheusConfig{
			Port:       2113,
			Timestamps: true,
			WindowSize: 8,
			Metrics: map[string]MetricConfig{
				"pcap_stats": MetricConfig{
					Aggregations: map[string]Aggregation{
						"ifname": Aggregation{
							Type: "EliminateDimension",
						},
					},
				},
				"second_ld_vs_rcode": MetricConfig{
					Aggregations: map[string]Aggregation{
						"SecondLD": Aggregation{
							Type: "MaxCells",
							Params: map[string]interface{}{
								"x": uint64(5),
							},
						},
					},
				},
				"priming_responses": MetricConfig{
					Aggregations: map[string]Aggregation{
						"ReplyLen": Aggregation{
							Type: "Bucket",
							Params: map[string]interface{}{
								"start": int64(-1),
								"width": uint64(50),
							},
						},
					},
				},
				"qr_aa_bits": MetricConfig{
					Aggregations: nil,
				},
				"qtype": MetricConfig{
					Aggregations: map[string]Aggregation{
						"Qtype": Aggregation{
							Type: "Filter",
							Params: map[string]interface{}{
								"A":    map[string]interface{}{},
								"AAAA": map[string]interface{}{},
								"NS":   nil,
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Logf("Parsed config: \n%+v\n\n", config)
		t.Logf("Expected config: \n%+v\n\n", expectedConfig)
		t.Errorf("Parsed config doesnt deeply match expected config")
	}
}
