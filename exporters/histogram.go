package exporters

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type HistogramValue struct {
	Buckets   map[float64]uint64
	Count     uint64
	Sum       float64
	Timestamp time.Time
}

func (v HistogramValue) GetMetric(labelValues LabelValues, desc *prometheus.Desc) prometheus.Metric {
	return prometheus.MustNewConstHistogram(
		desc,
		v.Count,
		v.Sum,
		v.Buckets,
		labelValues.ToList()...,
	)
}

func (v HistogramValue) GetTimestamp() time.Time {
	return v.Timestamp
}

type HistogramVec struct {
	*MetricVec[HistogramValue]
}

func NewHistogramVec(desc *prometheus.Desc) *HistogramVec {
	metricVec := NewMetricVec[HistogramValue](desc)
	return &HistogramVec{
		MetricVec: metricVec,
	}
}

func (h *HistogramVec) Add(labelValues LabelValues, bucketsIncrease map[float64]uint64, countIncrease uint64, sumIncrease float64, timestamp time.Time) {
	if sumIncrease < 0 {
		panic("Sum increase for histogram metric can't be smaller than 0")
	}
	newBuckets := bucketsIncrease
	newCount := countIncrease
	newSum := sumIncrease

	history := h.withLabelValues(labelValues)
	if !history.IsEmpty() {
		currentBuckets := history.Values[0].Buckets
		for le := range currentBuckets {
			newBuckets[le] += currentBuckets[le]
		}
		newCount += history.Values[0].Count
		newSum += history.Values[0].Sum
	}

	value := HistogramValue{
		Buckets:   newBuckets,
		Count:     newCount,
		Sum:       newSum,
		Timestamp: timestamp,
	}
	history.AddValue(value)
}
