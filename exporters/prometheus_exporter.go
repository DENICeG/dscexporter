package exporters

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DENICeG/dscexporter/aggregation"
	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Config *config.Config

func SetConf(config *config.Config) {
	Config = config
}

const NAMESERVER_LABEL = "ns"
const LOCATION_LABEL = "loc"

type MetricVec interface {
	Collect(ch chan<- prometheus.Metric)
	Describe(ch chan<- *prometheus.Desc)
}

type LabelValues struct {
	Location   string
	Nameserver string
	Label1     string
	Label2     string
}

func (me *LabelValues) LabelValues() []string {
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
	Metrics map[string]MetricVec
	mutex   sync.Mutex // Mutex to make sure that metrics are not queried while a DSC file gets parsed
}

func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		Metrics: make(map[string]MetricVec),
	}
}

func (pe *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pe, ch)
}

func (pe *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	for _, metric := range pe.Metrics {
		metric.Collect(ch)
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

func (pe *PrometheusExporter) CreateMissingHistogramVec(dataset *dscparser.Dataset) (histogramVec *HistogramVec, noneCounterVec *CounterVec) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	metricName := fmt.Sprintf("dsc_exporter_%v_%v", dataset.Name, dim2)

	histogramVecUncasted, ok := pe.Metrics[metricName]
	if ok {
		histogramVec = histogramVecUncasted.(*HistogramVec)
	} else {
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
		histogramVec = CreateHistogramVec(desc)
		pe.Metrics[metricName] = histogramVec
	}

	noneCounterMetricName := fmt.Sprintf("dsc_exporter_%v_%v_None", dataset.Name, dim2)
	noneCounterVecUncasted, ok := pe.Metrics[noneCounterMetricName]
	if ok {
		noneCounterVec = noneCounterVecUncasted.(*CounterVec)
	} else {
		noneCounterMetricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v for value None", dataset.Name, dim2)
		labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
		if dim1 != "All" {
			labels = append(labels, dim1)
		}
		noneCounterDesc := prometheus.NewDesc(
			noneCounterMetricName,
			noneCounterMetricHelp,
			labels,
			nil,
		)
		noneCounterVec = CreateCounterVec(noneCounterDesc)
		pe.Metrics[noneCounterMetricName] = noneCounterVec
	}
	return
}

func (pe *PrometheusExporter) ExportHistogram(dataset *dscparser.Dataset, metricConfig config.MetricConfig, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	histogramVec, noneCounterVec := pe.CreateMissingHistogramVec(dataset)
	_, params := metricConfig.IsBucket(dim2)

	for _, row := range dataset.Data.Rows {
		buckets, count, sum, noneCounter := CalculateBuckets(
			&row,
			float64(params.Start),
			float64(params.Width),
			float64(params.Count),
			dataset.Name,
		)

		labelVales := LabelValues{
			Location:   location,
			Nameserver: nameserver,
		}
		if dim1 != "All" {
			labelVales.Label1 = row.Value
		}

		timestamp := time.Unix(dataset.StopTime, 0)

		history := histogramVec.WithLabelValues(labelVales)
		history.Add(buckets, count, sum, timestamp)

		if noneCounter > 0 {
			noneCounterHistory := noneCounterVec.WithLabelValues(labelVales)
			noneCounterHistory.Add(noneCounter, timestamp)
		}
	}
}

func (pe *PrometheusExporter) CreateMissingCounterVec(dataset *dscparser.Dataset) *CounterVec {
	metricName := fmt.Sprintf("dsc_exporter_%v", dataset.Name)
	if counterVec, ok := pe.Metrics[metricName]; ok {
		return counterVec.(*CounterVec)
	}

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

	counterVec := CreateCounterVec(desc)
	pe.Metrics[metricName] = counterVec
	return counterVec
}

func (pe *PrometheusExporter) ExportCounter(dataset *dscparser.Dataset, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	counterVec := pe.CreateMissingCounterVec(dataset)

	for _, row := range dataset.Data.Rows {
		for _, cell := range row.Cells {

			labelValues := LabelValues{
				Location:   location,
				Nameserver: nameserver,
			}
			if dim1 != "All" {
				labelValues.Label1 = row.Value
			}
			if dim2 != "All" {
				labelValues.Label2 = cell.Value
			}

			valueHistory := counterVec.WithLabelValues(labelValues)
			valueHistory.Add(float64(cell.Count), time.Unix(dataset.StopTime, 0))
		}
	}
}

func (pe *PrometheusExporter) IncreaseParsedFiles(location string, nameserver string, timestamp int64) {
	metricName := "dsc_exporter_parsed_files"
	counterVecUncasted, ok := pe.Metrics[metricName]
	var counterVec *CounterVec
	if ok {
		counterVec = counterVecUncasted.(*CounterVec)
	} else {
		metricHelp := "How many files the dsc exporter parsed for each ns"
		labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
		desc := prometheus.NewDesc(
			metricName,
			metricHelp,
			labels,
			nil,
		)
		counterVec = CreateCounterVec(desc)
		pe.Metrics[metricName] = counterVec
	}

	labelValues := LabelValues{
		Location:   location,
		Nameserver: nameserver,
	}
	valueHistory := counterVec.WithLabelValues(labelValues)
	valueHistory.Add(1, time.Unix(timestamp, 0))
}

func (pe *PrometheusExporter) ExportDSCData(dscData *dscparser.DSCData) {

	if len(dscData.Datasets) == 0 {
		return
	}
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	aggregation.AggregateForPrometheus(dscData, *Config)

	for _, dataset := range dscData.Datasets {
		metricConfig, ok := Config.Prometheus.Metrics[dataset.Name]
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
}

func (pe *PrometheusExporter) StartPrometheusExporter() {

	slog.Info("Starting prometheus exporter", "url", fmt.Sprintf("http://localhost:%d/metrics", Config.Prometheus.Port))

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
	http.ListenAndServe(fmt.Sprintf(":%d", Config.Prometheus.Port), nil)
}
