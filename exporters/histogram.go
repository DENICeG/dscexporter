package exporters

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type TimestampedHistogramValues struct {
	Buckets   map[float64]uint64
	Count     uint64
	Sum       float64
	Timestamp time.Time
}

type HistogramValuesHistory struct {
	Values []TimestampedHistogramValues
}

func (h *HistogramValuesHistory) IsEmpty() bool {
	return len(h.Values) == 0
}

func (h *HistogramValuesHistory) checkValues(bucketsIncrease map[float64]uint64, countIncrease uint64, sumIncrease float64) {
	for _, inc := range bucketsIncrease {
		if inc < 0 {
			panic("Bucket increase for histogram metric can't be smaller than 0")
		}
	}
	if countIncrease < 0 {
		panic("Count increase for histogram metric can't be smaller than 0")
	}
	if sumIncrease < 0 {
		panic("Sum increase for histogram metric can't be smaller than 0")
	}
}

func (h *HistogramValuesHistory) Add(bucketsIncrease map[float64]uint64, countIncrease uint64, sumIncrease float64, timestamp time.Time) {
	h.checkValues(bucketsIncrease, countIncrease, sumIncrease)

	newBuckets := bucketsIncrease
	newCount := countIncrease
	newSum := sumIncrease

	if !h.IsEmpty() {
		currentBuckets := h.Values[0].Buckets
		for le := range currentBuckets {
			newBuckets[le] += currentBuckets[le]
		}
		newCount += h.Values[0].Count
		newSum += h.Values[0].Sum
	}

	timestampedValue := TimestampedHistogramValues{
		Buckets:   newBuckets,
		Count:     newCount,
		Sum:       newSum,
		Timestamp: timestamp,
	}
	h.Values = append(h.Values, timestampedValue)
	h.Values = h.Values[0:min(len(h.Values), Config.Prometheus.WindowSize)]
}

type HistogramVec struct {
	Values map[MetricIdentifier]HistogramValuesHistory
	Desc   *prometheus.Desc
}

func (h *HistogramVec) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(h, ch)
}

func (h *HistogramVec) CollectValue(ch chan<- prometheus.Metric, metricIdentifier MetricIdentifier, timestampedHistogramValues TimestampedHistogramValues) {
	labelValues := metricIdentifier.LabelValues()
	metric := prometheus.MustNewConstHistogram(
		h.Desc,
		timestampedHistogramValues.Count,
		timestampedHistogramValues.Sum,
		timestampedHistogramValues.Buckets,
		labelValues...,
	)
	if Config.Prometheus.Timestamps {
		metricWithTimesamp := prometheus.NewMetricWithTimestamp(timestampedHistogramValues.Timestamp, metric)
		ch <- metricWithTimesamp
	} else {
		ch <- metric
	}
}

func (h *HistogramVec) Collect(ch chan<- prometheus.Metric) {

	for metricIdentifier, valueHistory := range h.Values {
		if Config.Prometheus.Timestamps {
			for _, timestampedHistogramValues := range valueHistory.Values {
				h.CollectValue(ch, metricIdentifier, timestampedHistogramValues)
			}
		} else {
			if !valueHistory.IsEmpty() && Config.Prometheus.IsInTimeWindow(valueHistory.Values[0].Timestamp) {
				h.CollectValue(ch, metricIdentifier, valueHistory.Values[0])
			}
		}
	}
}

func (h *HistogramVec) WithLabelValues(metricIdentifier MetricIdentifier) (HistogramValuesHistory, bool) {
	history, ok := h.Values[metricIdentifier]
	return history, ok
}
