package scheduler

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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

type DSCFile struct {
	Location   string
	Nameserver string
	StopTime   time.Time
	FilePath   string
}

func ListDSCFiles(config config.Config) []DSCFile {

	dscFiles := make([]DSCFile, 0)

	fileNameRe, _ := regexp.Compile(`^(\d{10})\.dscdata\.xml$`)

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
			files, _ := os.ReadDir(nsFolderPath)

			//Sort dscfiles from oldest -> newest
			slices.SortFunc(files, func(a, b fs.DirEntry) int {
				return strings.Compare(a.Name(), b.Name())
			})

			for _, file := range files {

				matches := fileNameRe.FindStringSubmatch(file.Name())
				if !file.IsDir() && matches != nil {

					dscFilePath := filepath.Join(nsFolderPath, file.Name())

					stopTimeRaw := matches[1]
					stopTimeInt, err := strconv.Atoi(stopTimeRaw)
					checkError(err)
					stopTime := time.Unix(int64(stopTimeInt), 0)

					dscFiles = append(dscFiles, DSCFile{
						Location:   locationFolder.Name(),
						Nameserver: nsFolder.Name(),
						FilePath:   dscFilePath,
						StopTime:   stopTime,
					})
				}
			}

		}
	}
	return dscFiles
}

func ReadAndExportDir(config config.Config, exporter *exporters.PrometheusExporter) {

	dscFiles := ListDSCFiles(config)

	for _, dscFile := range dscFiles {

		exportStart := time.Now()
		dscData := dscparser.ReadFile(dscFile.FilePath, dscFile.Location, dscFile.Nameserver)

		exporter.ExportDSCData(dscData, dscFile.StopTime)
		slog.Info("Exported file",
			slog.String("nameserver", dscData.NameServer),
			slog.String("stop_time", dscFile.StopTime.String()),
			slog.String("delay", time.Since(dscFile.StopTime).String()),
			slog.String("took", time.Since(exportStart).String()),
		)
		//perfFile.WriteString(fmt.Sprintf("%v\n", time.Since(exportStart)))

		if config.RemoveReadFiles {
			err := os.Remove(dscFile.FilePath)
			checkError(err)
		}

		//perfFile.Sync()
	}

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
