package exporters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"
	"github.com/prometheus/client_golang/prometheus"

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

func TestCheckBucketBorders(t *testing.T) {
	row := dscparser.Row{}
	start := 31.0
	width := 32.0
	count := 100

	buckets, _, _, _ := CalculateBuckets(&row, start, width, float64(count), "ReplyLen")
	var bucketBorders []float64
	for le := range buckets {
		bucketBorders = append(bucketBorders, le)
	}
	sort.Float64s(bucketBorders)

	expectedBucketBorders := prometheus.LinearBuckets(start, width, count)
	assert.Equal(t, expectedBucketBorders, bucketBorders)
}

func TestCalculateBuckets(t *testing.T) {
	row := dscparser.ParseRow("./testdata/CalculateBuckets/row.xml")
	buckets, count, sum, noneCounter := CalculateBuckets(row, 50, 50, 5, "ReplyLen")

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

func TestPrometheusExporter(t *testing.T) {

	config := config.ParseConfig("./testdata/config.yaml")

	prometheusExporter := NewPrometheusExporter(config)

	go prometheusExporter.StartPrometheusExporter()

	//Export dsc file and check if its correctly exported
	dscData := dscparser.ReadFile("./testdata/test_dsc_file.xml", "loc", "ns")
	prometheusExporter.ExportDSCData(dscData)

	metrics := getMetrics(t, config)
	expected_metrics, err := os.ReadFile("./testdata/expected_metrics.txt")
	assert.NoError(t, err)
	assert.Equal(t, sortMetrics(string(expected_metrics)), sortMetrics(metrics))

	//Export another dsc file and check if its correctly exported too
	dscData2 := dscparser.ReadFile("./testdata/test_dsc_file2.xml", "loc", "ns")
	prometheusExporter.ExportDSCData(dscData2)

	metrics = getMetrics(t, config)
	expected_metrics, err = os.ReadFile("./testdata/expected_metrics2.txt")
	assert.NoError(t, err)
	assert.Equal(t, sortMetrics(string(expected_metrics)), sortMetrics(metrics))
}

func TestNewPrometheusExporter(t *testing.T) {
	config := config.ParseConfig("./testdata/config.yaml")
	//Shouldnt panic when creating multiple Exporters
	NewPrometheusExporter(config)
	NewPrometheusExporter(config)
}
