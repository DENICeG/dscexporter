package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/DENICeG/dscexporter/config"
	"github.com/DENICeG/dscexporter/exporters"
	"github.com/DENICeG/dscexporter/scheduler"

	"github.com/alecthomas/kingpin/v2"
)

var (
	gitcommit  string
	appversion string
	buildtime  string
)

var (
	app = kingpin.New("dsc-exporter", "A command-line tool to export DSC files.")
	//app.Version(fmt.Sprintf("app: %s - commit: %s - version: %s - buildtime: %s", app.Name, gitcommit, appversion, buildtime))
	configPath = app.Flag("config", "Path to the config file").Short('c').Envar("DSC_EXPORTER_CONFIG").Required().ExistingFile()
	data       = app.Flag("data", "Path to the data dir").Short('d').Envar("DSC_EXPORTER_DATADIR").ExistingDir()
	logLevel   = app.Flag("log-level", "The log level (\"debug\", \"info\", \"warn\", \"error\")").Short('l').Envar("DSC_EXPORTER_LOG_LEVEL").Enum("debug", "info", "warn", "error")
	interval   = app.Flag("interval", "The interval the exporter looks for new files").Short('i').Envar("DSC_EXPORTER_INTERVAL").Duration()
	port       = app.Flag("port", "The port under the prometheus metrics are served").Short('p').Envar("DSC_EXPORTER_PORT").Int()
	remove     = app.Flag("remove", "Remove read files").Bool()
)

func argsContain(args []string, substring string) bool {
	for _, arg := range args {
		if strings.Contains(arg, substring) {
			return true
		}
	}
	return false
}

func hasFlagSetShort(args []string, flag string, short string) bool {
	return argsContain(args, "--"+flag) || argsContain(args, "-"+short)
}

func hasFlagSet(args []string, flag string) bool {
	return argsContain(args, "--"+flag)
}

func ParamsToConfig(args []string) config.Config {
	kingpin.MustParse(app.Parse(args))
	conf := config.ParseConfig(*configPath)

	if hasFlagSetShort(args, "interval", "i") {
		conf.Interval = *interval
	}
	if hasFlagSetShort(args, "data", "d") {
		conf.DataDir = *data
	}
	if hasFlagSetShort(args, "port", "p") {
		conf.Prometheus.Port = *port
	}
	if hasFlagSetShort(args, "log-level", "l") {
		conf.LogLevel = config.GetLogLevel(*logLevel)
	}
	if hasFlagSet(args, "remove") || hasFlagSet(args, "no-remove") {
		conf.RemoveReadFiles = *remove
	}

	return conf
}

func main() {
	app.Version(fmt.Sprintf("app: %s - commit: %s - version: %s - buildtime: %s", app.Name, gitcommit, appversion, buildtime))

	conf := ParamsToConfig(os.Args[1:])

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: conf.LogLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Parsed config", "path", *configPath)

	prometheusExporter := exporters.NewPrometheusExporter(conf)
	go scheduler.Run(conf, prometheusExporter, scheduler.ReadAndExportDir)

	prometheusExporter.StartPrometheusExporter()
}
