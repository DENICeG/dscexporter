package scheduler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/dscparser"
	"github.com/DENICeG/dscexporter/exporters"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ReadAndExportDir(config config.Config, exporter *exporters.PrometheusExporter) {

	locationFolders, _ := os.ReadDir(config.DataDir)
	for _, locationFolder := range locationFolders {
		if !locationFolder.IsDir() {
			continue
		}

		locationFolderPath := filepath.Join(config.DataDir, locationFolder.Name())
		nsFolders, _ := os.ReadDir(locationFolderPath)

		for _, nsFolder := range nsFolders {
			if !nsFolder.IsDir() {
				continue
			}

			nsFolderPath := filepath.Join(locationFolderPath, nsFolder.Name())
			dscFiles, _ := os.ReadDir(nsFolderPath)

			for _, dscFile := range dscFiles {
				if !dscFile.IsDir() && strings.HasSuffix(dscFile.Name(), ".dscdata.xml") {

					exportStart := time.Now()

					dscFilePath := filepath.Join(nsFolderPath, dscFile.Name())
					dscData := dscparser.ReadFile(dscFilePath, locationFolder.Name(), nsFolder.Name())

					stopTimeRaw := dscData.Datasets[0].StopTime
					stopTime := time.Unix(int64(stopTimeRaw), 0)

					exporter.ExportDSCData(dscData)

					slog.Info("Exported file",
						slog.String("nameserver", dscData.NameServer),
						slog.Int64("stop_timestamp", stopTimeRaw),
						slog.String("stop_time", stopTime.String()),
						slog.String("delay", time.Since(stopTime).String()),
						slog.String("took", time.Since(exportStart).String()),
					)
					//perfFile.WriteString(fmt.Sprintf("%v\n", time.Since(exportStart)))

					if config.RemoveReadFiles {
						err := os.Remove(dscFilePath)
						checkError(err)
					}
				}
			}

		}
	}
	//perfFile.Sync()

}

func Run(config config.Config, exporter *exporters.PrometheusExporter, function func(config.Config, *exporters.PrometheusExporter)) {

	//f, _ := os.Create("perf.txt")
	//defer f.Sync()
	//defer f.Close()

	slog.Info("Started parsing dsc files", "path", config.DataDir)
	slog.Info("------------------------------------------------------------------")
	for i := 0; true; i++ {
		startTime := time.Now()

		function(config, exporter)

		endTime := time.Now()
		sleepDuration := max(config.Interval-endTime.Sub(startTime), 0)

		slog.Info("Done parsing data folder", "took", endTime.Sub(startTime), "sleeping_for", sleepDuration)
		slog.Info("------------------------------------------------------------------")
		time.Sleep(sleepDuration)
	}
}
