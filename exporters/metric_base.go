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
	if !h.IsEmpty() && h.Values[0].GetTimestamp().After(value.GetTimestamp()) {
		return // Out of order is discarded
	}
	h.Values = slices.Insert(h.Values, 0, value)
	if len(h.Values) > Config.Prometheus.WindowSize {
		h.Values = h.Values[:Config.Prometheus.WindowSize]
	}
}

type MetricVec[T MetricValue] struct {
	Values map[LabelValues]*WindowedHistory[T]
	Desc   *prometheus.Desc
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

			for i := len(history.Values) - 1; i >= 0; i-- {
				value := history.Values[i]
				// Lücken füllen
				if !Config.Prometheus.IsInTimeWindow(value.GetTimestamp()) {
					continue
				}

				if i+1 < len(history.Values) && Config.Prometheus.IsInTimeWindow(history.Values[i+1].GetTimestamp()) {
					olderValue := history.Values[i+1]
					diff := value.GetTimestamp().Sub(olderValue.GetTimestamp())
					minutesDiff := diff.Minutes()
					for j := 1.0; j < minutesDiff; j++ {
						gapTimestamp := olderValue.GetTimestamp().Add(time.Duration(j) * time.Minute)
						mv.collectValue(ch, labelValues, olderValue, gapTimestamp)
					}
				}
				mv.collectValue(ch, labelValues, value, value.GetTimestamp())
			}
		} else {
			if !history.IsEmpty() && Config.Prometheus.IsInTimeWindow(history.Values[0].GetTimestamp()) {
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
