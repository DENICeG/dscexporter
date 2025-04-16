package exporters

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusExporter struct {
	Metrics      map[string]interface{}
	FilesCounter *prometheus.CounterVec
	Registry     *prometheus.Registry
	Config       config.Config
}

func NewPrometheusExporter(config config.Config) *PrometheusExporter {
	registry := prometheus.NewRegistry()
	filesCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dsc_exporter_parsed_files",
			Help: "How many files the dsc exporter parsed for each ns",
		},
		[]string{"ns"},
	)
	registry.MustRegister(filesCounter)
	return &PrometheusExporter{Metrics: make(map[string]interface{}), Registry: registry, Config: config, FilesCounter: filesCounter}
}

func (pe *PrometheusExporter) addHistogram(metricName string, metricHelp string, buckets []float64, labels []string, datasetName string) {
	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metricName,
			Help:    metricHelp,
			Buckets: buckets,
		},
		labels,
	)
	pe.Registry.MustRegister(metric)
	pe.Metrics[datasetName] = metric
}

func (pe *PrometheusExporter) addCounter(metricName string, metricHelp string, labels []string, datasetName string) {
	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricName,
			Help: metricHelp,
		},
		labels,
	)
	pe.Registry.MustRegister(metric)
	pe.Metrics[datasetName] = metric
}

func (pe *PrometheusExporter) createMissingBucket(dataset dscparser.Dataset, metricConfig config.MetricConfig) {

	//First label is always nameserver
	var labels []string = []string{"ns"}
	var buckets []float64

	label1 := dataset.DimensionInfo[0].Type
	label2 := dataset.DimensionInfo[1].Type

	_, params := metricConfig.IsBucket(label2)
	start := float64(params.Start)
	width := float64(params.Width)
	count := params.Count
	buckets = prometheus.LinearBuckets(start, width, count)

	if label1 != "All" {
		labels = append(labels, label1)
	}

	metricName := fmt.Sprintf("dsc_exporter_%v", dataset.Name)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v", dataset.Name)
	pe.addHistogram(metricName, metricHelp, buckets, labels, dataset.Name)
}

func (pe *PrometheusExporter) createMissingCounter(dataset dscparser.Dataset) {
	var labels []string = []string{"ns"}
	for _, dimensionInfo := range dataset.DimensionInfo {
		label := dimensionInfo.Type
		if label != "All" {
			labels = append(labels, label)
		}
	}
	metricName := fmt.Sprintf("dsc_exporter_%v", dataset.Name)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v", dataset.Name)
	pe.addCounter(metricName, metricHelp, labels, dataset.Name)
}

func (pe *PrometheusExporter) createMissingMetrics(dscData *dscparser.DSCData) {

	for _, dataset := range dscData.Datasets {
		metricConfig, ok := pe.Config.Prometheus.Metrics[dataset.Name]
		if !ok {
			continue
		}
		if _, ok := pe.Metrics[dataset.Name]; ok {
			continue
		}

		// Only second dimension can be a bucket
		isBucket, _ := metricConfig.IsBucket(dataset.DimensionInfo[1].Type)
		if isBucket {
			pe.createMissingBucket(dataset, metricConfig)
		} else {
			pe.createMissingCounter(dataset)
		}

	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (pe *PrometheusExporter) ExportDataset(dataset *dscparser.Dataset, nameServer string) {
	metric := pe.Metrics[dataset.Name]
	metricConfig := pe.Config.Prometheus.Metrics[dataset.Name]

	label1 := dataset.DimensionInfo[0].Type
	label2 := dataset.DimensionInfo[1].Type

	for _, row := range dataset.Data.Rows {
		for _, cell := range row.Cells {

			//First label is always nameserver
			var labelValues []string = []string{nameServer}

			if label1 != "All" {
				labelValues = append(labelValues, row.Value)
			}
			if isBucket, _ := metricConfig.IsBucket(label2); label2 != "All" && !isBucket {
				labelValues = append(labelValues, cell.Value)
			}

			switch metricCasted := metric.(type) {
			case *prometheus.HistogramVec:

				bucket := 0.0
				cellValue, err := strconv.Atoi(cell.Value)
				checkError(err)
				bucket = float64(cellValue)

				for i := 0; i < cell.Count; i++ {
					metricCasted.WithLabelValues(labelValues...).Observe(bucket)
				}

			case *prometheus.CounterVec:
				metricCasted.WithLabelValues(labelValues...).Add(float64(cell.Count))
			default:
				fmt.Printf("Unkown metric type %T\n", metricCasted)
			}

		}

	}
}

func (pe *PrometheusExporter) ExportDSCData(dscData *dscparser.DSCData) {
	FilterForPrometheus(dscData, pe.Config)
	pe.createMissingMetrics(dscData)
	for _, dataset := range dscData.Datasets {
		if _, ok := pe.Metrics[dataset.Name]; !ok {
			continue
		}
		pe.ExportDataset(&dataset, dscData.NameServer)
	}
	pe.FilesCounter.WithLabelValues(dscData.NameServer).Inc()
}

func (pe *PrometheusExporter) StartPrometheusExporter() {

	//Disabled default go_collector exports for debuging and a better overview
	//ToDO: Enable later? Fix tests then...

	handler := promhttp.HandlerFor(pe.Registry, promhttp.HandlerOpts{})

	http.Handle("/metrics", handler)
	http.ListenAndServe(fmt.Sprintf(":%d", pe.Config.Prometheus.Port), nil)

}
