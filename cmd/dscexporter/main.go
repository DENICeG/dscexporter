package main

import (
	"log"
	"os"
	"slices"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/exporters"
	"github.com/DENICeG/dscexporter/scheduler"

	"github.com/alecthomas/kingpin/v2"
)

var (
	app = kingpin.New("dsc-exporter", "A command-line tool to export DSC files.")
	//app.Version(fmt.Sprintf("app: %s - commit: %s - version: %s - buildtime: %s", app.Name, gitcommit, appversion, buildtime))
	configPath = app.Flag("config", "Path to the config file").Short('c').Envar("DSC_EXPORTER_CONFIG").Required().ExistingFile()
	data       = app.Flag("data", "Path to the data dir").Short('d').Envar("DSC_EXPORTER_DATADIR").ExistingDir()
	interval   = app.Flag("interval", "The interval the exporter looks for new files").Short('i').Envar("DSC_EXPORTER_INTERVAL").Duration()
	port       = app.Flag("port", "The port under the prometheus metrics are served").Short('p').Envar("DSC_EXPORTER_PORT").Int()
	remove     = app.Flag("remove", "Remove read files").Bool()
)

func hasFlagSetShort(flag string, short rune) bool {
	return slices.Contains(os.Args, "--"+flag) || slices.Contains(os.Args, "-"+string(short))
}

func hasFlagSet(flag string) bool {
	return slices.Contains(os.Args, "--"+flag)
}

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	log.Printf("Parsing config %s", *configPath)
	config := config.ParseConfig(*configPath)

	if hasFlagSetShort("interval", 'i') {
		config.Interval = *interval
	}
	if hasFlagSetShort("data", 'd') {
		config.DataDir = *data
	}
	if hasFlagSetShort("port", 'p') {
		config.Prometheus.Port = *port
	}
	if hasFlagSet("remove") || hasFlagSet("no-remove") {
		config.RemoveReadFiles = *remove
	}

	prometheusExporter := exporters.NewPrometheusExporter(config)

	go scheduler.Run(config, prometheusExporter, scheduler.ReadAndExportDir)

	prometheusExporter.StartPrometheusExporter()
}
