package exporters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"

	"github.com/stretchr/testify/assert"
)

func getMetrics(t *testing.T, config config.Config) string {

	url := fmt.Sprintf("http://localhost:%d/metrics", config.Prometheus.Port)

	resp, err := http.Get(url)
	assert.NoError(t, err, "Unexpected error")

	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, 200)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Unexpected error")

	return string(body)
}

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
	assert.Equal(t, metrics, string(expected_metrics))
	t.Log(metrics)

	//Export another dsc file and check if its correctly exported too
	dscData2 := dscparser.ReadFile("./testdata/test_dsc_file2.xml", "loc", "ns")
	prometheusExporter.ExportDSCData(dscData2)

	metrics = getMetrics(t, config)
	expected_metrics, err = os.ReadFile("./testdata/expected_metrics2.txt")
	assert.NoError(t, err)
	assert.Equal(t, metrics, string(expected_metrics))
}

func TestNewPrometheusExporter(t *testing.T) {
	config := config.ParseConfig("./testdata/config.yaml")
	//Shouldnt panic when creating multiple Exporters
	NewPrometheusExporter(config)
	NewPrometheusExporter(config)
}
