package exporters

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type CounterValue struct {
	Value     float64
	Timestamp time.Time
}

func (v CounterValue) GetMetric(labelValues LabelValues, desc *prometheus.Desc) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		desc,
		prometheus.CounterValue,
		v.Value,
		labelValues.ToList()...,
	)
}

func (v CounterValue) GetTimestamp() time.Time {
	return v.Timestamp
}

type CounterVec struct {
	*MetricVec[CounterValue]
}

func NewCounterVec(desc *prometheus.Desc) *CounterVec {
	metricVec := NewMetricVec[CounterValue](desc)
	return &CounterVec{
		MetricVec: metricVec,
	}
}

func (c *CounterVec) Add(labelValues LabelValues, increase float64, timestamp time.Time) {

	if increase < 0 {
		panic("Increase for counter metric can't be smaller than 0")
	}

	history := c.withLabelValues(labelValues)

	newValue := increase
	if !history.IsEmpty() {
		newValue += history.Values[0].Value
	}

	value := CounterValue{
		Value:     newValue,
		Timestamp: timestamp,
	}
	history.AddValue(value)
}
