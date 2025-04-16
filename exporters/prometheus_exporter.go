package exporters

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

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

func (pe *PrometheusExporter) addHistogram(metricName string, metricHelp string, buckets []float64, labels []string, key string) {
	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metricName,
			Help:    metricHelp,
			Buckets: buckets,
		},
		labels,
	)
	pe.Registry.MustRegister(metric)
	pe.Metrics[key] = metric
}

func (pe *PrometheusExporter) addCounter(metricName string, metricHelp string, labels []string, key string) {
	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metricName,
			Help: metricHelp,
		},
		labels,
	)
	pe.Registry.MustRegister(metric)
	pe.Metrics[key] = metric
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

	metricName := fmt.Sprintf("dsc_exporter_%v_%v", dataset.Name, label2)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v", dataset.Name, label2)
	if params.UseMidpoint {
		metricHelp += " - DO NOT use the _sum value! This metric is based of a ranges in the dsc files, so the _sum value cant be calculated correctly"
	}
	pe.addHistogram(metricName, metricHelp, buckets, labels, dataset.Name)

	if params.NoneCounter {
		metricName := fmt.Sprintf("dsc_exporter_%v_%v_None", dataset.Name, label2)
		metricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v for value None", dataset.Name, label2)
		pe.addCounter(metricName, metricHelp, labels, fmt.Sprintf("%v_%v", dataset.Name, "None"))
	}
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

func (pe *PrometheusExporter) updateBucket(dataset *dscparser.Dataset, metricConfig config.MetricConfig, metric *prometheus.HistogramVec, label2 string, labelValues []string, value string, count int) {

	_, bucketParams := metricConfig.IsBucket(label2)
	if value == "None" && bucketParams.NoneCounter {
		// Increment counter for none values
		noneCounter := pe.Metrics[fmt.Sprintf("%v_%v", dataset.Name, "None")].(*prometheus.CounterVec)
		noneCounter.WithLabelValues(labelValues...).Add(float64(count))
		return
	}
	bucket := float64(0)
	if _, params := metricConfig.IsBucket(label2); strings.Contains(value, "-") && params.UseMidpoint {
		// For existing dsc ranges like 1024-1535 in EDNSBufSiz use midpoint
		substrings := strings.Split(value, "-")
		start, err1 := strconv.Atoi(substrings[0])
		end, err2 := strconv.Atoi(substrings[1])
		if err1 != nil || err2 != nil {
			panic(fmt.Sprintf("Value %v of dataset %v cant be splited and parsed for bucket", value, dataset.Name))
		}
		bucket = (float64(end) + float64(start)) / 2
	} else {
		cellValue, err := strconv.Atoi(value)
		checkError(err)
		bucket = float64(cellValue)
	}

	for i := 0; i < count; i++ {
		metric.WithLabelValues(labelValues...).Observe(bucket)
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
				pe.updateBucket(dataset, metricConfig, metricCasted, label2, labelValues, cell.Value, cell.Count)
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
