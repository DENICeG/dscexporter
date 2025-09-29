package exporters

import (
	"testing"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func prepareHistgramVec() (histogramVec *HistogramVec, labelValues LabelValues) {
	desc := prometheus.NewDesc(
		"metric",
		"help",
		[]string{LOCATION_LABEL, NAMESERVER_LABEL},
		nil,
	)
	buckets := []float64{10, 20}
	histogramVec = NewHistogramVec(desc, buckets)

	labelValues = LabelValues{
		Location:   "loc",
		Nameserver: "ns",
	}
	return
}

func TestAddToHistogramHistory(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	histogramVec, labelValues := prepareHistgramVec()

	lastFullMin := config.GetLastFullMin()
	histogramVec.Add(labelValues, map[float64]uint64{10: 10, 20: 20}, 30, 600, lastFullMin.Add(-3*time.Minute))
	histogramVec.Add(labelValues, map[float64]uint64{10: 10, 20: 20}, 30, 600, lastFullMin.Add(-2*time.Minute))
	histogramVec.Add(labelValues, map[float64]uint64{10: 0, 20: 1}, 1, 20, lastFullMin.Add(-1*time.Minute))
	histogramVec.Add(labelValues, map[float64]uint64{10: 1, 20: 1}, 1, 10, lastFullMin)

	history := histogramVec.withLabelValues(labelValues)
	assert.Equal(t,
		[]HistogramValue{{
			Buckets:   map[float64]uint64{10: 21, 20: 42},
			Count:     62,
			Sum:       1230,
			Timestamp: lastFullMin,
		}, {
			Buckets:   map[float64]uint64{10: 20, 20: 41},
			Count:     61,
			Sum:       1220,
			Timestamp: lastFullMin.Add(-1 * time.Minute),
		}, {
			Buckets:   map[float64]uint64{10: 20, 20: 40},
			Count:     60,
			Sum:       1200,
			Timestamp: lastFullMin.Add(-2 * time.Minute),
		}},
		history.Values,
	)
}

func TestAddBadBucketsToHistogramHistory(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	histogramVec, labelValues := prepareHistgramVec()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	lastFullMin := config.GetLastFullMin()
	histogramVec.Add(labelValues, map[float64]uint64{10: 10, 20: 20}, 30, 600, lastFullMin.Add(-1*time.Minute))
	histogramVec.Add(labelValues, map[float64]uint64{20: 10}, 10, 200, lastFullMin)
}

func TestCollectHistogramVec(t *testing.T) {
	SetConf(&config.Config{
		Prometheus: config.PrometheusConfig{
			Timestamps: true,
			WindowSize: 3,
		},
	})
	histogramVec, labelValues := prepareHistgramVec()

	lastFullMin := config.GetLastFullMin()
	histogramVec.Add(labelValues, map[float64]uint64{10: 10, 20: 20}, 30, 600, lastFullMin.Add(-1*time.Minute))
	histogramVec.Add(labelValues, map[float64]uint64{10: 10, 20: 20}, 30, 600, lastFullMin)

	ch := make(chan prometheus.Metric)
	go func() {
		histogramVec.Collect(ch)
		close(ch)
	}()

	metrics := collectValues(ch)
	assert.Equal(t, 2, len(metrics))
}
