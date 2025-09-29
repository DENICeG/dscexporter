package exporters

import (
	"context"
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

type PrometheusExporter struct {
	Counters   map[string]*CounterVec
	Histograms map[string]*HistogramVec
	mutex      sync.Mutex // Mutex to make sure that metrics are not queried while a DSC file gets parsed
	Server     *http.Server
}

func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		Counters:   make(map[string]*CounterVec),
		Histograms: make(map[string]*HistogramVec),
	}
}

func (pe *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pe, ch)
}

func (pe *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	for _, metric := range pe.Counters {
		metric.Collect(ch)
	}
	for _, metric := range pe.Histograms {
		metric.Collect(ch)
	}
}

func (pe *PrometheusExporter) CreateCounter(metricName string, metricHelp string, labels []string) *CounterVec {
	desc := prometheus.NewDesc(
		metricName,
		metricHelp,
		labels,
		nil,
	)
	counterVec := NewCounterVec(desc)
	pe.Counters[metricName] = counterVec
	return counterVec
}

func (pe *PrometheusExporter) CreateHistogram(metricName string, metricHelp string, labels []string, buckets []float64) *HistogramVec {
	desc := prometheus.NewDesc(
		metricName,
		metricHelp,
		labels,
		nil,
	)
	histogramVec := NewHistogramVec(desc, buckets)
	pe.Histograms[metricName] = histogramVec
	return histogramVec
}

func CalculateBuckets(row *dscparser.Row, params config.BucketParams, datasetName string) (buckets map[float64]uint64, count uint64, sum float64, noneCounter float64) {

	buckets = make(map[float64]uint64)
	for _, le := range params.Buckets() {
		buckets[le] = 0
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

		for _, le := range params.Buckets() {
			if value <= le {
				buckets[le] += uint64(cell.Count)
			}
		}
	}

	return //Named return values are returned
}

func (pe *PrometheusExporter) CreateMissingHistogramVec(dataset *dscparser.Dataset, params config.BucketParams) (histogramVec *HistogramVec, noneCounterVec *CounterVec) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	metricName := fmt.Sprintf("dsc_exporter_%v_%v", dataset.Name, dim2)

	histogramVec, ok := pe.Histograms[metricName]
	if !ok {
		metricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v", dataset.Name, dim2)
		labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
		if dim1 != "All" {
			labels = append(labels, dim1)
		}

		histogramVec = pe.CreateHistogram(metricName, metricHelp, labels, params.Buckets())
	}

	noneCounterMetricName := fmt.Sprintf("dsc_exporter_%v_%v_None", dataset.Name, dim2)
	noneCounterVec, ok = pe.Counters[noneCounterMetricName]
	if !ok {
		noneCounterMetricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v for value None", dataset.Name, dim2)
		labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
		if dim1 != "All" {
			labels = append(labels, dim1)
		}
		noneCounterVec = pe.CreateCounter(noneCounterMetricName, noneCounterMetricHelp, labels)
	}
	return
}

func (pe *PrometheusExporter) ExportHistogram(dataset *dscparser.Dataset, metricConfig config.MetricConfig, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type
	_, params := metricConfig.IsBucket(dim2)
	histogramVec, noneCounterVec := pe.CreateMissingHistogramVec(dataset, params)

	for _, row := range dataset.Data.Rows {
		buckets, count, sum, noneCounter := CalculateBuckets(
			&row,
			params,
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

		histogramVec.Add(labelVales, buckets, count, sum, timestamp)
		if noneCounter > 0 {
			noneCounterVec.Add(labelVales, noneCounter, timestamp)
		}
	}
}

func (pe *PrometheusExporter) CreateMissingCounterVec(dataset *dscparser.Dataset) *CounterVec {
	metricName := fmt.Sprintf("dsc_exporter_%v", dataset.Name)
	if counterVec, ok := pe.Counters[metricName]; ok {
		return counterVec
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
	return pe.CreateCounter(metricName, metricHelp, labels)
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
			counterVec.Add(labelValues, float64(cell.Count), time.Unix(dataset.StopTime, 0))
		}
	}
}

func (pe *PrometheusExporter) IncreaseParsedFiles(location string, nameserver string, timestamp time.Time) {
	metricName := "dsc_exporter_parsed_files"
	counterVec, ok := pe.Counters[metricName]
	if !ok {
		metricHelp := "How many files the dsc exporter parsed for each ns"
		labels := []string{LOCATION_LABEL, NAMESERVER_LABEL}
		counterVec = pe.CreateCounter(metricName, metricHelp, labels)
	}

	labelValues := LabelValues{
		Location:   location,
		Nameserver: nameserver,
	}
	counterVec.Add(labelValues, 1, timestamp)
}

func (pe *PrometheusExporter) ExportDSCData(dscData *dscparser.DSCData, stopTime time.Time) {
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
	pe.IncreaseParsedFiles(dscData.Location, dscData.NameServer, stopTime)
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

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	pe.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", Config.Prometheus.Port),
		Handler: mux,
	}

	go func() {
		if err := pe.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Prometheus Handler error", "error", err)
		}
	}()

	time.Sleep(1 * time.Second)
}

func (pe *PrometheusExporter) ShutdownPrometheusExporter() {
	slog.Info("Shutting down prometheus exporter")
	if err := pe.Server.Shutdown(context.Background()); err != nil {
		slog.Error("Prometheus Handler error while shutting down", "error", err)
	}
}
