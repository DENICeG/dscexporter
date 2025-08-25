package exporters

import (
	"testing"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func prepareCounterVec() (counterVec *CounterVec, labelValues LabelValues) {
	desc := prometheus.NewDesc(
		"metric",
		"help",
		[]string{LOCATION_LABEL, NAMESERVER_LABEL},
		nil,
	)
	counterVec = NewCounterVec(desc)

	labelValues = LabelValues{
		Location:   "loc",
		Nameserver: "ns",
	}
	return
}

func collectValues(ch <-chan prometheus.Metric) []prometheus.Metric {
	metrics := make([]prometheus.Metric, 0)
	for metric := range ch {
		metrics = append(metrics, metric)
	}
	return metrics
}

func TestAddToCounterHistory(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	counterVec, labelValues := prepareCounterVec()

	now := time.Now()
	counterVec.Add(labelValues, 10, now.Add(-3*time.Minute))
	counterVec.Add(labelValues, 10, now.Add(-2*time.Minute))
	counterVec.Add(labelValues, 10, now.Add(-1*time.Minute))
	counterVec.Add(labelValues, 10, now)

	history := counterVec.withLabelValues(labelValues)
	assert.Equal(t,
		[]CounterValue{{
			Value:     40,
			Timestamp: now,
		}, {
			Value:     30,
			Timestamp: now.Add(-1 * time.Minute),
		}, {
			Value:     20,
			Timestamp: now.Add(-2 * time.Minute),
		}},
		history.Values,
	)
}

func TestCollectCounter(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	counterVec, labelValues := prepareCounterVec()
	now := time.Now()
	counterVec.Add(labelValues, 10, now.Add(-1*time.Minute))
	counterVec.Add(labelValues, 10, now)

	ch := make(chan prometheus.Metric)
	go func() {
		counterVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 2, len(metrics))
}

func TestCollectToOld(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: false,
			WindowSize: 3,
		},
	})

	counterVec, labelValues := prepareCounterVec()
	now := time.Now()
	counterVec.Add(labelValues, 10, now.Add(-4*time.Minute)) // Metric older than 3 Minutes (configured via WindowSize), so its not collected
	counterVec.Add(labelValues, 10, now.Add(-2*time.Minute))

	ch := make(chan prometheus.Metric)
	go func() {
		counterVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 1, len(metrics))
}
