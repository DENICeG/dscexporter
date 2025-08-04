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

const NAMESERVER_LABEL = "ns"
const LOCATION_LABEL = "loc"

type MetricIdentifier struct {
	DatasetName string
	Location    string
	Nameserver  string
	Label1      string
	Label2      string
}

type PrometheusExporter struct {
	MetricsRing   [][]prometheus.Metric //Ring buffer for metrics # Geht nicht... Zwischenspeicher
	CurrentValues map[MetricIdentifier]any
	windowSize    int
	start         int
	Config        config.Config
}

func (pe *PrometheusExporter) AddCounter(metricIdentifier MetricIdentifier, desc *prometheus.Desc, increase float64, timestamp int64) {
	newValue := increase
	if currentValue, ok := pe.CurrentValues[metricIdentifier].(float64); ok {
		newValue += currentValue
	}
	pe.CurrentValues[metricIdentifier] = newValue

	labelValues := []string{metricIdentifier.Location, metricIdentifier.Nameserver}
	if metricIdentifier.Label1 != "" {
		labelValues = append(labelValues, metricIdentifier.Label1)
	}
	if metricIdentifier.Label2 != "" {
		labelValues = append(labelValues, metricIdentifier.Label2)
	}

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

func (pe *PrometheusExporter) AddHistogram(metricIdentifier MetricIdentifier, desc *prometheus.Desc, buckets map[float64]uint64, count uint64, sum float64, timestamp int64) {

	if currentBuckets, ok := pe.CurrentValues[metricIdentifier].(map[float64]uint64); ok {
		for le := range buckets {
			buckets[le] += currentBuckets[le]
		}
	}
	pe.CurrentValues[metricIdentifier] = buckets

	countIdentifier := metricIdentifier
	countIdentifier.DatasetName = fmt.Sprintf("%v_count", countIdentifier.DatasetName)
	if currentCount, ok := pe.CurrentValues[countIdentifier].(uint64); ok {
		count += currentCount
	}
	pe.CurrentValues[countIdentifier] = count

	sumIdentifier := metricIdentifier
	sumIdentifier.DatasetName = fmt.Sprintf("%v_sum", sumIdentifier.DatasetName)
	if currentSum, ok := pe.CurrentValues[sumIdentifier].(float64); ok {
		sum += currentSum
	}
	pe.CurrentValues[sumIdentifier] = sum

	labelValues := []string{metricIdentifier.Location, metricIdentifier.Nameserver}
	if metricIdentifier.Label1 != "" {
		labelValues = append(labelValues, metricIdentifier.Label1)
	}

	metric := prometheus.MustNewConstHistogram(
		desc,
		count,
		sum,
		buckets,
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
	if !config.Prometheus.Timestamps {
		windowSize = 1
	}
	return &PrometheusExporter{
		MetricsRing:   make([][]prometheus.Metric, 0),
		CurrentValues: make(map[MetricIdentifier]any),
		windowSize:    windowSize,
		start:         -1,
		Config:        config,
	}
}

func (pe *PrometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pe, ch)
}

func (pe *PrometheusExporter) Collect(ch chan<- prometheus.Metric) {
	if pe.start < 0 {
		return // No metrics yet
	}

	for i := 0; i < min(len(pe.MetricsRing)-pe.start, pe.windowSize); i++ {
		metricsAtTimestamp := pe.MetricsRing[i+pe.start]
		for _, metric := range metricsAtTimestamp {
			ch <- metric
		}
	}
}

func (pe *PrometheusExporter) CalculateBuckets(row *dscparser.Row, bucketStart float64, bucketWidth float64, bucketCount float64, datasetName string) (buckets map[float64]uint64, count uint64, sum float64, noneCounter float64) {

	buckets = make(map[float64]uint64)

	for i := 0.0; i < bucketCount; i++ {
		buckets[bucketStart+bucketWidth*i] = 0
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

		bucket := float64(int((bucketStart+value)/bucketWidth))*bucketWidth + bucketStart // Calculate value for the histogram le label
		if bucket <= bucketStart+bucketCount*bucketWidth {                                //  Check that the label is unter maximum le label. Bigger labels are represented by the +inf label
			buckets[bucket] += uint64(cell.Count)
		}

	}

	return //Named return values are returned
}

func (pe *PrometheusExporter) ExportHistogram(dataset *dscparser.Dataset, metricConfig config.MetricConfig, location string, nameserver string) {

	dim1 := dataset.DimensionInfo[0].Type
	dim2 := dataset.DimensionInfo[1].Type

	_, params := metricConfig.IsBucket(dim2)

	metricName := fmt.Sprintf("dsc_exporter_%v_%v", dataset.Name, dim2)
	metricHelp := fmt.Sprintf("DSC-Metric from dataset %v for %v", dataset.Name, dim2)
	if params.UseMidpoint {
		metricHelp += " - DO NOT use the _sum value! This metric is based of a ranges in the dsc files, so the _sum value cant be calculated correctly"
	}

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

	allBuckets := make(map[float64]uint64)
	allCount := uint64(0)
	allSum := 0.0
	allNoneCounter := 0.0

	for _, row := range dataset.Data.Rows {

		buckets, count, sum, noneCounter := pe.CalculateBuckets(
			&row,
			float64(params.Start),
			float64(params.Width),
			float64(params.Count),
			dataset.Name,
		)

		if dim1 != "All" {
			metricIdentifier := MetricIdentifier{
				DatasetName: dataset.Name,
				Location:    location,
				Nameserver:  nameserver,
				Label1:      row.Value,
			}

			pe.AddHistogram(metricIdentifier, desc, buckets, count, sum, dataset.StopTime)

			if noneCounter > 0 {
				noneCounterIdentifier := metricIdentifier
				noneCounterIdentifier.DatasetName = fmt.Sprintf("%v_none", noneCounterIdentifier.DatasetName)
				pe.AddCounter(metricIdentifier, noneCounterDesc, noneCounter, dataset.StopTime)
			}
		} else {
			for le := range buckets {
				if _, ok := allBuckets[le]; !ok {
					allBuckets[le] = 0
				}
				allBuckets[le] += buckets[le]
			}
			allCount += count
			allSum += sum
			allNoneCounter += noneCounter
		}
	}

	if dim1 == "All" {
		metricIdentifier := MetricIdentifier{
			DatasetName: dataset.Name,
			Location:    location,
			Nameserver:  nameserver,
		}

		pe.AddHistogram(metricIdentifier, desc, allBuckets, allCount, allSum, dataset.StopTime)
		if allNoneCounter > 0 {
			noneCounterIdentifier := metricIdentifier
			noneCounterIdentifier.DatasetName = fmt.Sprintf("%v_none", noneCounterIdentifier.DatasetName)
			pe.AddCounter(metricIdentifier, noneCounterDesc, allNoneCounter, dataset.StopTime)
		}
	}
}

func (pe *PrometheusExporter) ExportCounter(dataset *dscparser.Dataset, location string, nameserver string) {
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

	for _, row := range dataset.Data.Rows {
		for _, cell := range row.Cells {

			metricIdentifier := MetricIdentifier{
				DatasetName: dataset.Name,
				Location:    location,
				Nameserver:  nameserver,
			}

			labelValues := []string{location, nameserver}
			if dim1 != "All" {
				labelValues = append(labelValues, row.Value)
				metricIdentifier.Label1 = row.Value
			}
			if dim2 != "All" {
				labelValues = append(labelValues, cell.Value)
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
		DatasetName: "parsed_files",
		Location:    location,
		Nameserver:  nameserver,
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

	registry := prometheus.NewPedanticRegistry()
	//registry := prometheus.NewRegistry()
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
