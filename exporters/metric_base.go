package exporters

import (
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type LabelValues struct {
	Location   string
	Nameserver string
	Label1     string
	Label2     string
}

func (me *LabelValues) ToList() []string {
	labelValues := []string{me.Location, me.Nameserver}
	if me.Label1 != "" {
		labelValues = append(labelValues, me.Label1)
	}
	if me.Label2 != "" {
		labelValues = append(labelValues, me.Label2)
	}
	return labelValues
}

type MetricValue interface {
	GetMetric(labelValues LabelValues, desc *prometheus.Desc) prometheus.Metric
	GetTimestamp() time.Time
}

type WindowedHistory[T MetricValue] struct {
	Values []T
}

func (h *WindowedHistory[T]) IsEmpty() bool {
	return len(h.Values) == 0
}

func (h *WindowedHistory[T]) AddValue(value T) {
	if !h.IsEmpty() && (h.Values[0].GetTimestamp().After(value.GetTimestamp()) || h.Values[0].GetTimestamp().Equal(value.GetTimestamp())) {
		return // Out of order is ignored
	}
	h.Values = slices.Insert(h.Values, 0, value)
	maxLength := max(Config.Prometheus.WindowSize, 1)
	if len(h.Values) > maxLength {
		h.Values = h.Values[:maxLength]
	}
}

func (h *WindowedHistory[T]) FindValueAtTimestamp(timestamp time.Time) int {
	for i, value := range h.Values {
		if value.GetTimestamp().Before(timestamp) || value.GetTimestamp().Equal(timestamp) {
			return i
		}
	}
	return -1
}

type MetricVec[T MetricValue] struct {
	Values map[LabelValues]*WindowedHistory[T] // These values store the real values with the real timestamps.
	// If there is no value for a specific timestamp in the dscfile, there will be no value in this array.
	// This way the code could check whether a value has been refreshed in the last x minutes, to possibly stop exporting the value to minimize the number of active time series.
	Desc *prometheus.Desc
}

func NewMetricVec[T MetricValue](desc *prometheus.Desc) *MetricVec[T] {
	return &MetricVec[T]{
		Values: make(map[LabelValues]*WindowedHistory[T]),
		Desc:   desc,
	}
}

func (mv *MetricVec[T]) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(mv, ch)
}

func (mv *MetricVec[T]) collectValue(ch chan<- prometheus.Metric, labelValues LabelValues, value T, timestamp time.Time) {
	metric := value.GetMetric(labelValues, mv.Desc)
	if Config.Prometheus.Timestamps {
		ch <- prometheus.NewMetricWithTimestamp(timestamp, metric)
	} else {
		ch <- metric
	}
}

func (mv *MetricVec[T]) Collect(ch chan<- prometheus.Metric) {
	for labelValues, history := range mv.Values {
		if Config.Prometheus.Timestamps {
			/*
				Collects the values of the history.
				It does it by going from the newest timestamp back in the history every minute.
				It is not enough to just collect the values in the Values array.
				These values contain gaps if an time series has not been increased by a dscfile.
				So this code looks at every timestamp in the TimeWindow and exports the closest value to the timestamp.
			*/
			if history.IsEmpty() {
				continue
			}
			newestTimestamp := history.Values[0].GetTimestamp()
			diffInMinutes := int(time.Since(newestTimestamp).Minutes())
			for i := 0; i < Config.Prometheus.WindowSize-diffInMinutes; i++ {
				timestamp := newestTimestamp.Add(-time.Duration(i) * time.Minute)
				value_index := history.FindValueAtTimestamp(timestamp)
				if value_index >= 0 {
					value := history.Values[value_index]
					mv.collectValue(ch, labelValues, value, timestamp)
				}
			}
		} else {
			if !history.IsEmpty() {
				mv.collectValue(ch, labelValues, history.Values[0], history.Values[0].GetTimestamp())
			}
		}
	}
}

func (mv *MetricVec[T]) withLabelValues(labels LabelValues) *WindowedHistory[T] {
	h, ok := mv.Values[labels]
	if !ok {
		h = &WindowedHistory[T]{Values: make([]T, 0)}
		mv.Values[labels] = h
	}
	return h
}
