package exporters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"

	"github.com/stretchr/testify/assert"
)

func getMetrics(t *testing.T, config config.Config) string {

	url := fmt.Sprintf("http://localhost:%d/metrics", config.Prometheus.Port)

	resp, err := http.Get(url)
	assert.NoError(t, err, "Unexpected error")

	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Unexpected error")

	return string(body)
}

func sortMetrics(metrics string) string {
	lines := strings.Split(metrics, "\n")
	slices.SortFunc(lines, strings.Compare)
	return strings.Join(lines, "\n")
}

func TestCalculateBuckets(t *testing.T) {
	row := dscparser.ParseRow("./testdata/CalculateBuckets/row.xml")
	params := config.BucketParams{
		Start: 50,
		Width: 50,
		Count: 5,
	}
	buckets, count, sum, noneCounter := CalculateBuckets(row, params, "ReplyLen")

	expectedBuckets := map[float64]uint64{
		50:  25,
		100: 40,
		150: 45,
		200: 45,
		250: 65,
	}
	assert.EqualValues(t, expectedBuckets, buckets)
	assert.Equal(t, uint64(70), count)
	assert.Equal(t, 9115.0, sum)
	assert.Equal(t, 5.0, noneCounter)
}

// func TestAddCounter(t *testing.T) {

// 	desc := prometheus.NewDesc(
// 		"metric_name",
// 		"help",
// 		[]string{"loc", "ns", "label1", "label2"},
// 		nil,
// 	)
// 	metricIdentifier := MetricIdentifier{
// 		DatasetName: "test_dataset",
// 		Location:    "loc",
// 		Nameserver:  "ns",
// 		Label1:      "label_value_1",
// 		Label2:      "label_value_2",
// 	}
// 	AddCounter(metricIdentifier, desc, 5.0, 1755506438)
// }

func TestPrometheusExporterWithoutTimestamps(t *testing.T) {

	conf := config.ParseConfig("./testdata/NoTimestamp/config.yaml")
	SetConf(&conf)

	prometheusExporter := NewPrometheusExporter()
	prometheusExporter.StartPrometheusExporter()
	defer prometheusExporter.ShutdownPrometheusExporter()

	lastFullMin := config.GetLastFullMin()

	//Export dsc file and check if its correctly exported
	dscData := dscparser.ReadFileWithCustomTimestamp("./testdata/NoTimestamp/test_dsc_file.xml", "loc", "ns", lastFullMin.Add(-time.Minute))
	prometheusExporter.ExportDSCData(dscData, lastFullMin.Add(-1*time.Minute))

	metrics := getMetrics(t, conf)
	expected_metrics, err := os.ReadFile("./testdata/NoTimestamp/expected_metrics.txt")
	assert.NoError(t, err)
	assert.Equal(t, string(expected_metrics), metrics)

	//Export another dsc file and check if its correctly exported too
	dscData2 := dscparser.ReadFileWithCustomTimestamp("./testdata/NoTimestamp/test_dsc_file2.xml", "loc", "ns", lastFullMin)
	prometheusExporter.ExportDSCData(dscData2, lastFullMin)

	metrics = getMetrics(t, conf)
	expected_metrics, err = os.ReadFile("./testdata/NoTimestamp/expected_metrics2.txt")
	assert.NoError(t, err)
	assert.Equal(t, string(expected_metrics), metrics)
}

func TestPrometheusExporterWithTimestamps(t *testing.T) {

	conf := config.ParseConfig("./testdata/Timestamp/config.yaml")
	SetConf(&conf)

	prometheusExporter := NewPrometheusExporter()
	prometheusExporter.StartPrometheusExporter()
	defer prometheusExporter.ShutdownPrometheusExporter()

	lastFullMin := config.GetLastFullMin()

	//DSC File to old, so its not collected
	dscData := dscparser.ReadFileWithCustomTimestamp("./testdata/Timestamp/pcap_and_priming_queries.xml", "loc", "ns", lastFullMin.Add(-3*time.Minute))
	prometheusExporter.ExportDSCData(dscData, lastFullMin.Add(-3*time.Minute))

	dscData = dscparser.ReadFileWithCustomTimestamp("./testdata/Timestamp/only_pcap.xml", "loc", "ns", lastFullMin.Add(-2*time.Minute))
	prometheusExporter.ExportDSCData(dscData, lastFullMin.Add(-2*time.Minute))

	dscData = dscparser.ReadFileWithCustomTimestamp("./testdata/Timestamp/only_pcap.xml", "loc", "ns", lastFullMin.Add(-1*time.Minute))
	prometheusExporter.ExportDSCData(dscData, lastFullMin.Add(-1*time.Minute))

	dscData = dscparser.ReadFileWithCustomTimestamp("./testdata/Timestamp/pcap_and_priming_queries.xml", "loc", "ns", lastFullMin)
	prometheusExporter.ExportDSCData(dscData, lastFullMin)

	metrics := getMetrics(t, conf)
	// fmt.Println("Real:")
	// fmt.Println(metrics)

	expected_metrics_bytes, err := os.ReadFile("./testdata/Timestamp/expected_metrics.txt")
	assert.NoError(t, err)
	expected_metrics := string(expected_metrics_bytes)
	expected_metrics = strings.ReplaceAll(expected_metrics, "[now]", fmt.Sprintf("%d", lastFullMin.UnixMilli()))
	expected_metrics = strings.ReplaceAll(expected_metrics, "[-1m]", fmt.Sprintf("%d", lastFullMin.Add(-1*time.Minute).UnixMilli()))
	expected_metrics = strings.ReplaceAll(expected_metrics, "[-2m]", fmt.Sprintf("%d", lastFullMin.Add(-2*time.Minute).UnixMilli()))

	// fmt.Println()
	// fmt.Println("Expected:")
	// fmt.Println(expected_metrics)

	assert.Equal(t, expected_metrics, metrics)
}

func TestNewPrometheusExporter(t *testing.T) {
	config := config.ParseConfig("./testdata/NoTimestamp/config.yaml")
	SetConf(&config)
	//Shouldnt panic when creating multiple Exporters
	exporter := NewPrometheusExporter()
	exporter.StartPrometheusExporter()
	exporter.ShutdownPrometheusExporter()

	exporter2 := NewPrometheusExporter()
	exporter2.StartPrometheusExporter()
	exporter2.ShutdownPrometheusExporter()
}
