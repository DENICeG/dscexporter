package exporters

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DENICeG/dscexporter/aggregation"
	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Config *config.Config

const NAMESERVER_LABEL = "ns"
const LOCATION_LABEL = "loc"

type MetricIdentifier struct {
	MetricName string
	Location   string
	Nameserver string
	Label1     string
	Label2     string
}

func (me *MetricIdentifier) LabelValues() []string {
	labelValues := []string{me.Location, me.Nameserver}
	if me.Label1 != "" {
		labelValues = append(labelValues, me.Label1)
	}
	if me.Label2 != "" {
		labelValues = append(labelValues, me.Label2)
	}
	return labelValues
}

type PrometheusExporter struct {
	MetricsRing          [][]prometheus.Metric //Ring buffer for metrics
	CurrentValuesUInt    map[MetricIdentifier]uint64
	CurrentValuesFloat   map[MetricIdentifier]float64
	CurrentValuesBuckets map[MetricIdentifier]map[float64]uint64
	windowSize           int
	start                int
	Config               config.Config
}

func (pe *PrometheusExporter) IncreaseCurrentFlaotValue(metricIdentifier MetricIdentifier, increase float64) float64 {
	newValue := increase
	if currentValue, ok := pe.CurrentValuesFloat[metricIdentifier]; ok {
		newValue += currentValue
	}
	pe.CurrentValuesFloat[metricIdentifier] = newValue
	return newValue
}

func (pe *PrometheusExporter) IncreaseCurrentUIntValue(metricIdentifier MetricIdentifier, increase uint64) uint64 {
	newValue := increase
	if currentValue, ok := pe.CurrentValuesUInt[metricIdentifier]; ok {
		newValue += currentValue
	}
	pe.CurrentValuesUInt[metricIdentifier] = newValue
	return newValue
}

func (pe *PrometheusExporter) IncreaseCurrentBucketsValue(metricIdentifier MetricIdentifier, bucketsIncrease map[float64]uint64) map[float64]uint64 {
	newBuckets := bucketsIncrease
	if currentBuckets, ok := pe.CurrentValuesBuckets[metricIdentifier]; ok {
		for le := range currentBuckets {
			newBuckets[le] += currentBuckets[le]
		}
	}
	pe.CurrentValuesBuckets[metricIdentifier] = newBuckets
	return newBuckets
}

func (pe *PrometheusExporter) AddCounter(metricIdentifier MetricIdentifier, desc *prometheus.Desc, increase float64, timestamp int64) {
	newValue := pe.IncreaseCurrentFlaotValue(metricIdentifier, increase)
	labelValues := metricIdentifier.LabelValues()

	metric := prometheus.MustNewConstMetric(
		desc,
		prometheus.CounterValue,
		newValue,
		labelValues...,
	)
	if pe.Config.Prometheus.Timestamps {
		metricWithTimesamp := prometheus.NewMetricWithTimestamp(time.Unix(timestamp, 0), metric)
		pe.MetricsRing[0] = append(pe.MetricsRing[0], metricWithTimesamp)
	} else {
		pe.MetricsRing[0] = append(pe.MetricsRing[0], metric)
	}

}

func (pe *PrometheusExporter) AddHistogram(metricIdentifier MetricIdentifier, desc *prometheus.Desc, bucketsIncrease map[float64]uint64, countIncrease uint64, sumIncrease float64, timestamp int64) {

	newBuckets := pe.IncreaseCurrentBucketsValue(metricIdentifier, bucketsIncrease)

	countIdentifier := metricIdentifier
	countIdentifier.MetricName = fmt.Sprintf("%v_count", metricIdentifier.MetricName)
	newCount := pe.IncreaseCurrentUIntValue(countIdentifier, countIncrease)

	sumIdentifier := metricIdentifier
	sumIdentifier.MetricName = fmt.Sprintf("%v_sum", metricIdentifier.MetricName)
	newSum := pe.IncreaseCurrentFlaotValue(metricIdentifier, sumIncrease)

	labelValues := metricIdentifier.LabelValues()

	metric := prometheus.MustNewConstHistogram(
		desc,
		newCount,
		newSum,
		newBuckets,
		labelValues...,
	)

	if pe.Config.Prometheus.Timestamps {
		metricWithTimesamp := prometheus.NewMetricWithTimestamp(time.Unix(timestamp, 0), metric)
		pe.MetricsRing[0] = append(pe.MetricsRing[0], metricWithTimesamp)
	} else {
		pe.MetricsRing[0] = append(pe.MetricsRing[0], metric)
	}
}

func (pe *PrometheusExporter) NewTimestamp() {
	pe.MetricsRing = slices.Insert(pe.MetricsRing, 0, make([]prometheus.Metric, 0))
	pe.start = 1
}

func (pe *PrometheusExporter) ExportTimestamp() {
	pe.start = 0
	pe.MetricsRing = pe.MetricsRing[0:min(len(pe.MetricsRing), pe.windowSize)]
}

func NewPrometheusExporter(config config.Config) *PrometheusExporter {
	windowSize := config.Prometheus.WindowSize
	// if !config.Prometheus.Timestamps {
	// 	windowSize = 1
	// }
	return &PrometheusExporter{
		MetricsRing:          make([][]prometheus.Metric, 0),
		CurrentValuesFloat:   make(map[MetricIdentifier]float64),
		CurrentValuesUInt:    make(map[MetricIdentifier]uint64),
		CurrentValuesBuckets: make(map[MetricIdentifier]map[float64]uint64),
		windowSize:           windowSize,
		start:                0,
		Config:               config,
	}
}

func (pe *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pe, ch)
}

func (pe *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {

	//exportedMetrics := make(map[int]bool)

	for i := pe.start; i < min(len(pe.MetricsRing), pe.start+pe.windowSize); i++ {
		metricsAtTimestamp := pe.MetricsRing[i]
		for _, metric := range metricsAtTimestamp {
			fmt.Println(metric.Desc().String())
			ch <- metric
		}
	}
}

func CalculateBuckets(row *dscparser.Row, bucketStart float64, bucketWidth float64, bucketCount float64, datasetName string) (buckets map[float64]uint64, count uint64, sum float64, noneCounter float64) {

	buckets = make(map[float64]uint64)

	for i := 0.0; i < bucketCount; i++ {
		buckets[bucketStart+bucketWidth*i] = 0 // The key is the bucket border (the le label)
	}

	count = uint64(0)
	sum = 0.0
	noneCounter = 0

	for _, cell := range row.Cells {

		if cell.Value == "None" {
			noneCounter += float64(cell.Count)
			continue
		}

		value := 0.0
		if strings.Contains(cell.Value, "-") {
			// For existing dsc ranges like 1024-1535 in EDNSBufSiz use midpoint
			substrings := strings.Split(cell.Value, "-")
			start, err1 := strconv.Atoi(substrings[0])
			end, err2 := strconv.Atoi(substrings[1])
			if err1 != nil || err2 != nil {
				panic(fmt.Sprintf("Value %v of dataset %v cant be splited and parsed for bucket", cell.Value, datasetName))
			}
			value = (float64(end) + float64(start)) / 2
		} else {
			cellValue, err := strconv.Atoi(cell.Value)
			if err != nil {
				panic(fmt.Sprintf("Value %v of dataset %v cant be parsed for bucket", cell.Value, datasetName))
			}
			value = float64(cellValue)
		}

		count += uint64(cell.Count)
		sum += float64(cell.Count) * value

		for i := 0.0; i < bucketCount; i++ {
			le := bucketStart + bucketWidth*i
			if value <= le {
				buckets[le] += uint64(cell.Count)
			}
		}
	}

	return //Named return values are returned
}

func (pe *PrometheusExporter) CreateHistogramDesc(dataset *dscparser.Dataset) (string, *prometheus.Desc, string, *prometheus.Desc) {
	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type

	metricName := fmt.Sprintf("dsc_exporter_%v_%v", dataset.Name, dim2)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v", dataset.Name, dim2)

	labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
	if dim1 != "All" {
		labels = append(labels, dim1)
	}
	desc := prometheus.NewDesc(
		metricName,
		metricHelp,
		labels,
		nil,
	)

	noneCounterMetricName := fmt.Sprintf("dsc_exporter_%v_%v_None", dataset.Name, dim2)
	noneCounterMetricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v for value None", dataset.Name, dim2)
	noneCounterDesc := prometheus.NewDesc(
		noneCounterMetricName,
		noneCounterMetricHelp,
		labels,
		nil,
	)
	return metricName, desc, noneCounterMetricName, noneCounterDesc
}

func (pe *PrometheusExporter) ExportHistogram(dataset *dscparser.Dataset, metricConfig config.MetricConfig, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	metricName, desc, noneCounterMetricName, noneCounterDesc := pe.CreateHistogramDesc(dataset)
	_, params := metricConfig.IsBucket(dim2)

	for _, row := range dataset.Data.Rows {
		buckets, count, sum, noneCounter := CalculateBuckets(
			&row,
			float64(params.Start),
			float64(params.Width),
			float64(params.Count),
			dataset.Name,
		)

		metricIdentifier := MetricIdentifier{
			MetricName: metricName,
			Location:   location,
			Nameserver: nameserver,
		}
		if dim1 != "All" {
			metricIdentifier.Label1 = row.Value
		}

		pe.AddHistogram(metricIdentifier, desc, buckets, count, sum, dataset.StopTime)

		if noneCounter > 0 {
			noneCounterIdentifier := metricIdentifier
			noneCounterIdentifier.MetricName = noneCounterMetricName
			pe.AddCounter(metricIdentifier, noneCounterDesc, noneCounter, dataset.StopTime)
		}
	}
}

func (pe *PrometheusExporter) CreateCounterDesc(dataset *dscparser.Dataset) (string, *prometheus.Desc) {
	metricName := fmt.Sprintf("dsc_exporter_%v", dataset.Name)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v", dataset.Name)

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type

	labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
	if dim1 != "All" {
		labels = append(labels, dim1)
	}
	if dim2 != "All" {
		labels = append(labels, dim2)
	}

	desc := prometheus.NewDesc(
		metricName,
		metricHelp,
		labels,
		nil,
	)
	return metricName, desc
}

func (pe *PrometheusExporter) ExportCounter(dataset *dscparser.Dataset, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	metricName, desc := pe.CreateCounterDesc(dataset)

	for _, row := range dataset.Data.Rows {
		for _, cell := range row.Cells {

			metricIdentifier := MetricIdentifier{
				MetricName: metricName,
				Location:   location,
				Nameserver: nameserver,
			}
			if dim1 != "All" {
				metricIdentifier.Label1 = row.Value
			}
			if dim2 != "All" {
				metricIdentifier.Label2 = cell.Value
			}

			pe.AddCounter(metricIdentifier, desc, float64(cell.Count), dataset.StopTime)
		}
	}
}

func (pe *PrometheusExporter) IncreaseParsedFiles(location string, nameserver string, timestamp int64) {
	metricName := "dsc_exporter_parsed_files"
	metricHelp := "How many files the dsc exporter parsed for each ns"
	labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}

	desc := prometheus.NewDesc(
		metricName,
		metricHelp,
		labels,
		nil,
	)
	metricIdentifier := MetricIdentifier{
		MetricName: metricName,
		Location:   location,
		Nameserver: nameserver,
	}

	pe.AddCounter(metricIdentifier, desc, 1.0, timestamp)
}

func (pe *PrometheusExporter) ExportDSCData(dscData *dscparser.DSCData) {

	if len(dscData.Datasets) == 0 {
		return
	}

	aggregation.AggregateForPrometheus(dscData, pe.Config)

	pe.NewTimestamp()

	for _, dataset := range dscData.Datasets {
		metricConfig, ok := pe.Config.Prometheus.Metrics[dataset.Name]
		if !ok {
			continue
		}
		label2 := dataset.DimensionInfo[1].Type
		if isBucket, _ := metricConfig.IsBucket(label2); isBucket {
			pe.ExportHistogram(&dataset, metricConfig, dscData.Location, dscData.NameServer)
		} else {
			pe.ExportCounter(&dataset, dscData.Location, dscData.NameServer)
		}
	}

	timestamp := dscData.Datasets[0].StopTime
	pe.IncreaseParsedFiles(dscData.Location, dscData.NameServer, timestamp)
	pe.ExportTimestamp()
}

func (pe *PrometheusExporter) StartPrometheusExporter() {

	slog.Info("Starting prometheus exporter", "url", fmt.Sprintf("http://localhost:%d/metrics", pe.Config.Prometheus.Port))

	//registry := prometheus.NewPedanticRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(pe)

	//Disabled default go_collector exports for debuging and a better overview
	//ToDO: Enable later? Fix tests then...
	// reg.MustRegister(
	// 	prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	// 	prometheus.NewGoCollector(),
	// )

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	http.Handle("/metrics", handler)
	http.ListenAndServe(fmt.Sprintf(":%d", pe.Config.Prometheus.Port), nil)
}
