package scheduler

import (
	"fmt"
	"log"
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

					dscFilePath := filepath.Join(nsFolderPath, dscFile.Name())
					dscData := dscparser.ReadFile(dscFilePath, locationFolder.Name(), nsFolder.Name())

					stopTime := dscData.Datasets[0].StopTime
					stopTimeUnix := time.Unix(int64(stopTime), 0)
					log.Printf("Exporting %s - %d (%s)", dscData.NameServer, stopTime, stopTimeUnix)

					exporter.ExportDSCData(dscData)

					if config.RemoveReadFiles {
						err := os.Remove(dscFilePath)
						checkError(err)
					}
				}
			}

		}

	}

}

func Run(config config.Config, exporter *exporters.PrometheusExporter, function func(config.Config, *exporters.PrometheusExporter)) {

	log.Printf("Started parsing dsc files in folder: %s", config.DataDir)
	for i := 0; true; i++ {
		startTime := time.Now()

		function(config, exporter)

		endTime := time.Now()
		sleepDuration := max(config.Interval-endTime.Sub(startTime), 0)

		log.Printf("Parsing took: %v, sleeping for: %v", endTime.Sub(startTime), sleepDuration)
		time.Sleep(sleepDuration)
	}
}
