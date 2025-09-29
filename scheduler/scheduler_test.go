package scheduler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/exporters"

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

	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Unexpected error")

	return string(body)
}

func copyDSCFiles(t *testing.T) {

	os.RemoveAll("./testdata/tmp/")
	conf := config.ParseConfigText([]byte("data: ./testdata/dsc-data"))
	dscFiles := ListDSCFiles(conf)

	lastFullMin := config.GetLastFullMin()
	newestTestData := time.Unix(1741170600, 0)
	for _, dscFile := range dscFiles {

		fileContent, err := os.ReadFile(dscFile.FilePath)
		assert.NoError(t, err)
		content := string(fileContent)

		diff := newestTestData.Sub(dscFile.StopTime)
		newStopTime := lastFullMin.Add(-diff)

		content = strings.ReplaceAll(
			content,
			fmt.Sprintf("start_time=\"%d\"", dscFile.StopTime.Unix()-60),
			fmt.Sprintf("start_time=\"%d\"", newStopTime.Unix()-60),
		)
		content = strings.ReplaceAll(
			content,
			fmt.Sprintf("stop_time=\"%d\"", dscFile.StopTime.Unix()),
			fmt.Sprintf("stop_time=\"%d\"", newStopTime.Unix()),
		)

		newFilePath := fmt.Sprintf("./testdata/tmp/%s/%s/%d.dscdata.xml", dscFile.Location, dscFile.Nameserver, newStopTime.Unix())
		dir := filepath.Dir(newFilePath)
		err = os.MkdirAll(dir, os.ModePerm)
		assert.NoError(t, err)

		file, err := os.Create(newFilePath)
		assert.NoError(t, err)
		defer file.Close()

		file.WriteString(content)
		file.Sync()
	}
}

func TestReadAndExportDir(t *testing.T) {
	copyDSCFiles(t)

	dscFilesBeforeRun := ListDSCFiles(config.ParseConfigText([]byte("data: ./testdata/tmp")))
	assert.Len(t, dscFilesBeforeRun, 4)

	conf := config.ParseConfig("./testdata/config.yml")
	exporters.SetConf(&conf)

	prometheusExporter := exporters.NewPrometheusExporter()
	prometheusExporter.StartPrometheusExporter()
	defer prometheusExporter.ShutdownPrometheusExporter()

	ReadAndExportDir(conf, prometheusExporter)

	metrics := getMetrics(t, conf)

	expectedMetrics, err := os.ReadFile("./testdata/expected_metrics.txt")
	assert.NoError(t, err)

	assert.Equal(t, string(expectedMetrics), metrics)

	dscFilesAfterRun := ListDSCFiles(config.ParseConfigText([]byte("data: ./testdata/tmp")))
	assert.Len(t, dscFilesAfterRun, 0)
}

func TestSortFilesList(t *testing.T) {
	files, err := os.ReadDir("./testdata/dsc-data/loc1/ns-1.loc1.de/")
	assert.NoError(t, err)

	SortFilesList(files)
	assert.Equal(t, "1741170540.dscdata.xml", files[0].Name())
	assert.Equal(t, "1741170600.dscdata.xml", files[1].Name())
}
