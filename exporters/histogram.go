package exporters

import (
	"fmt"
	"slices"
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
	Buckets []float64 // all le values, except +inf
}

func NewHistogramVec(desc *prometheus.Desc, buckets []float64) *HistogramVec {
	metricVec := NewMetricVec[HistogramValue](desc)
	slices.Sort(buckets)
	return &HistogramVec{
		MetricVec: metricVec,
		Buckets:   buckets,
	}
}

func (h *HistogramVec) Add(labelValues LabelValues, bucketsIncrease map[float64]uint64, countIncrease uint64, sumIncrease float64, timestamp time.Time) {
	if sumIncrease < 0 {
		panic("Sum increase for histogram metric can't be smaller than 0")
	}

	newBuckets := make(map[float64]uint64) // Only use in h.Bucket defined buckets
	lastValue := uint64(0)
	for _, le := range h.Buckets {
		v, ok := bucketsIncrease[le]
		if !ok {
			panic(fmt.Sprintf("Missing bucket %v", le))
		}
		if v < lastValue {
			panic(fmt.Sprintf("Invalid bucket %v, bucket value is smaller than a bucket before", le))
		}
		newBuckets[le] = bucketsIncrease[le]
		lastValue = v
	}
	newCount := countIncrease
	newSum := sumIncrease

	history := h.withLabelValues(labelValues)
	if !history.IsEmpty() {
		currentBuckets := history.Values[0].Buckets
		for _, le := range h.Buckets {
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
