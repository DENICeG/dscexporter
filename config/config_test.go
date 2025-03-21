package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsBucket(t *testing.T) {
	config := ParseConfig("./testdata/config.yaml")
	metricConfig := config.Prometheus.Metrics["priming_responses"]
	assert.True(t, metricConfig.IsBucket("ReplyLen"))
}

func TestConfig(t *testing.T) {

	config := ParseConfig("./testdata/config.yaml")

	expectedConfig := Config{
		RemoveReadFiles: true,
		DataDir:         DefaultDataDir,
		Interval:        20 * time.Second,
		Prometheus: PrometheusConfig{
			Port: 2113,
			Metrics: map[string]MetricConfig{
				"pcap_stats": MetricConfig{
					Aggregations: nil,
				},
				"second_ld_vs_rcode": MetricConfig{
					Aggregations: map[string]Aggregation{
						"SecondLD": Aggregation{
							Type: "MaxCells",
							Params: map[string]int{
								"x": 5,
							},
						},
					},
				},
				"priming_responses": MetricConfig{
					Aggregations: map[string]Aggregation{
						"ReplyLen": Aggregation{
							Type: "Bucket",
							Params: map[string]int{
								"start": 0,
								"width": 50,
								"count": 22,
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
