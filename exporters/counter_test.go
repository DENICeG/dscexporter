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

func TestFindValueBeforeTimestamp(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	counterVec, labelValues := prepareCounterVec()
	lastFullMin := config.GetLastFullMin()
	counterVec.Add(labelValues, 10, lastFullMin.Add(-1*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin)

	valueIndex := counterVec.withLabelValues(labelValues).FindValueAtTimestamp(lastFullMin)
	assert.Equal(t, 0, valueIndex)

	valueIndex = counterVec.withLabelValues(labelValues).FindValueAtTimestamp(lastFullMin.Add(-30 * time.Second))
	assert.Equal(t, 1, valueIndex)

	valueIndex = counterVec.withLabelValues(labelValues).FindValueAtTimestamp(lastFullMin.Add(-61 * time.Second))
	assert.Equal(t, -1, valueIndex)
}

func TestAddToCounterHistory(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	counterVec, labelValues := prepareCounterVec()

	lastFullMin := config.GetLastFullMin()
	counterVec.Add(labelValues, 10, lastFullMin.Add(-2*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin.Add(-3*time.Minute)) // Export out of order, so this data is ignored
	counterVec.Add(labelValues, 10, lastFullMin.Add(-1*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin)

	history := counterVec.withLabelValues(labelValues)
	assert.Equal(t,
		[]CounterValue{{
			Value:     30,
			Timestamp: lastFullMin,
		}, {
			Value:     20,
			Timestamp: lastFullMin.Add(-1 * time.Minute),
		}, {
			Value:     10,
			Timestamp: lastFullMin.Add(-2 * time.Minute),
		}},
		history.Values,
	)
}

func TestCollectCounter(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 5,
		},
	})
	counterVec, labelValues := prepareCounterVec()
	lastFullMin := config.GetLastFullMin()
	counterVec.Add(labelValues, 10, lastFullMin.Add(-5*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin.Add(-1*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin)

	ch := make(chan prometheus.Metric)
	go func() {
		counterVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 5, len(metrics))
}

func TestCollectOnlyUpToNewestTimestamp(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 5,
		},
	})

	counterVec, labelValues := prepareCounterVec()
	lastFullMin := config.GetLastFullMin()
	counterVec.Add(labelValues, 10, lastFullMin.Add(-3*time.Minute))
	counterVec.Add(labelValues, 10, lastFullMin.Add(-2*time.Minute))
	// No value for -1m and now should be collected

	ch := make(chan prometheus.Metric)
	go func() {
		counterVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 2, len(metrics))
}

func TestCollectTimestampsDisabled(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: false,
			WindowSize: 3,
		},
	})

	counterVec, labelValues := prepareCounterVec()
	lastFullMin := config.GetLastFullMin()
	counterVec.Add(labelValues, 10, lastFullMin.Add(-3*time.Minute))
	//Should just collect latest value, even if he is out of the time window

	ch := make(chan prometheus.Metric)
	go func() {
		counterVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 1, len(metrics))
}
