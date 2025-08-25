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

func (mv *MetricVec[T]) collectValue(ch chan<- prometheus.Metric, labelValues LabelValues, value T) {
	metric := value.GetMetric(labelValues, mv.Desc)
	if Config.Prometheus.Timestamps {
		ch <- prometheus.NewMetricWithTimestamp(value.GetTimestamp(), metric)
	} else {
		ch <- metric
	}
}

func (mv *MetricVec[T]) Collect(ch chan<- prometheus.Metric) {
	for labelValues, history := range mv.Values {
		if Config.Prometheus.Timestamps {
			for _, value := range history.Values {
				if Config.Prometheus.IsInTimeWindow(value.GetTimestamp()) {
					mv.collectValue(ch, labelValues, value)
				}
			}
		} else {
			if !history.IsEmpty() && Config.Prometheus.IsInTimeWindow(history.Values[0].GetTimestamp()) {
				mv.collectValue(ch, labelValues, history.Values[0])
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
