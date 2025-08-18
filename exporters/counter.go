package exporters

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type TimestampedValue struct {
	Value     float64
	Timestamp time.Time
}

type CounterValueHistory struct {
	Values []TimestampedValue
}

func (h *CounterValueHistory) IsEmpty() bool {
	return len(h.Values) == 0
}

func (h *CounterValueHistory) Add(increase float64, timestamp time.Time) {
	if increase < 0 {
		panic("Increase for counter metric can't be smaller than 0")
	}

	newValue := increase
	if !h.IsEmpty() {
		newValue += h.Values[0].Value
	}

	timestampedValue := TimestampedValue{
		Value:     newValue,
		Timestamp: timestamp,
	}
	h.Values = append(h.Values, timestampedValue)
	h.Values = h.Values[0:min(len(h.Values), Config.Prometheus.WindowSize)]
}

type CounterVec struct {
	Values map[LabelValues]CounterValueHistory
	Desc   *prometheus.Desc
}

func CreateCounterVec(desc *prometheus.Desc) *CounterVec {
	return &CounterVec{
		Values: make(map[LabelValues]CounterValueHistory),
		Desc:   desc,
	}
}

func (c *CounterVec) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func (c *CounterVec) CollectValue(ch chan<- prometheus.Metric, labelValues LabelValues, timestampedValue TimestampedValue) {
	metric := prometheus.MustNewConstMetric(
		c.Desc,
		prometheus.CounterValue,
		timestampedValue.Value,
		labelValues.LabelValues()...,
	)
	if Config.Prometheus.Timestamps {
		metricWithTimesamp := prometheus.NewMetricWithTimestamp(timestampedValue.Timestamp, metric)
		ch <- metricWithTimesamp
	} else {
		ch <- metric
	}
}

func (c *CounterVec) Collect(ch chan<- prometheus.Metric) {

	for labelValues, valueHistory := range c.Values {
		if Config.Prometheus.Timestamps {
			for _, timestampedValue := range valueHistory.Values {
				c.CollectValue(ch, labelValues, timestampedValue)
			}
		} else {
			if !valueHistory.IsEmpty() && Config.Prometheus.IsInTimeWindow(valueHistory.Values[0].Timestamp) {
				c.CollectValue(ch, labelValues, valueHistory.Values[0])
			}
		}
	}
}

func (c *CounterVec) WithLabelValues(labelValues LabelValues) CounterValueHistory {
	history, ok := c.Values[labelValues]
	if !ok {
		history = CounterValueHistory{
			Values: make([]TimestampedValue, 0),
		}
		c.Values[labelValues] = history
	}
	return history
}
