package scheduler

import (
	"dscexporter/config"
	"dscexporter/exporters"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testRunForSetup(t *testing.T, interval time.Duration, sleep time.Duration) {
	config_ := config.ParseConfigText([]byte("interval: " + interval.String()))

	channel := make(chan int)
	timeout1 := func(config config.Config, exporter *exporters.PrometheusExporter) {
		time.Sleep(sleep)
		channel <- 0
	}

	startTime := time.Now()
	go Run(config_, nil, timeout1)

	_, _ = <-channel, <-channel //Wait until timeout has run 2 times

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	expeced_duration := max(config_.Interval, sleep) + sleep
	assert.GreaterOrEqual(t, duration, expeced_duration)
	assert.LessOrEqual(t, duration, expeced_duration+100*time.Millisecond)
}

func TestRun0Interval(t *testing.T) {
	testRunForSetup(t, 0, 1*time.Second)
}

func TestRunShortInterval(t *testing.T) {
	testRunForSetup(t, 500*time.Millisecond, 1*time.Second)
}

func TestRunNormalInterval(t *testing.T) {
	testRunForSetup(t, 2*time.Second, 1*time.Second)
}

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

func TestReadAndExportDir(t *testing.T) {
	config := config.ParseConfigText([]byte("data: ./testdata/dsc-data\nremove: false"))

	prometheusExporter := exporters.NewPrometheusExporter(config)
	go prometheusExporter.StartPrometheusExporter()

	ReadAndExportDir(config, prometheusExporter)

	metrics := getMetrics(t, config)

	assert.Contains(t, metrics, `dsc_exporter_parsed_files{ns="ns-1.loc1.de"} 2`)
	assert.Contains(t, metrics, `dsc_exporter_parsed_files{ns="ns-2.loc1.de"} 1`)
	assert.Contains(t, metrics, `dsc_exporter_parsed_files{ns="ns-1.loc2.de"} 1`)
}
