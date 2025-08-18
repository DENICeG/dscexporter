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
	Values map[LabelValues]*HistogramValuesHistory
	Desc   *prometheus.Desc
}

func CreateHistogramVec(desc *prometheus.Desc) *HistogramVec {
	return &HistogramVec{
		Values: make(map[LabelValues]*HistogramValuesHistory),
		Desc:   desc,
	}
}

func (h *HistogramVec) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(h, ch)
}

func (h *HistogramVec) CollectValue(ch chan<- prometheus.Metric, labelValues LabelValues, timestampedHistogramValues TimestampedHistogramValues) {
	metric := prometheus.MustNewConstHistogram(
		h.Desc,
		timestampedHistogramValues.Count,
		timestampedHistogramValues.Sum,
		timestampedHistogramValues.Buckets,
		labelValues.LabelValues()...,
	)
	if Config.Prometheus.Timestamps {
		metricWithTimesamp := prometheus.NewMetricWithTimestamp(timestampedHistogramValues.Timestamp, metric)
		ch <- metricWithTimesamp
	} else {
		ch <- metric
	}
}

func (h *HistogramVec) Collect(ch chan<- prometheus.Metric) {

	for labelValues, valueHistory := range h.Values {
		if Config.Prometheus.Timestamps {
			for _, timestampedHistogramValues := range valueHistory.Values {
				h.CollectValue(ch, labelValues, timestampedHistogramValues)
			}
		} else {
			if !valueHistory.IsEmpty() && Config.Prometheus.IsInTimeWindow(valueHistory.Values[0].Timestamp) {
				h.CollectValue(ch, labelValues, valueHistory.Values[0])
			}
		}
	}
}

func (h *HistogramVec) WithLabelValues(labelValues LabelValues) *HistogramValuesHistory {
	history, ok := h.Values[labelValues]
	if !ok {
		history = &HistogramValuesHistory{
			Values: make([]TimestampedHistogramValues, 0),
		}
		h.Values[labelValues] = history
	}
	return history
}
