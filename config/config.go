package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

const DefaultInterval = 5 * time.Second
const DefaultDataDir = "/data/exporter_dsc"
const DefaultRemoveReadFiles = false
const DefaultPrometheusPort = 2112
const DefaultLogLevel = slog.LevelInfo
const DefaultTimestamps = true
const DefaultWindowSize = 5

type Config struct {
	RemoveReadFiles bool             `yaml:"remove"`
	DataDir         string           `yaml:"data"`
	Interval        time.Duration    `yaml:"interval"`
	Prometheus      PrometheusConfig `yaml:"prometheus"`
	LogLevel        slog.Level       `yaml:"loglevel"`
	//Database DatabaseConfig `yaml:"database"`
}

type PrometheusConfig struct {
	Metrics    map[string]MetricConfig `yaml:"metrics"`
	Port       int                     `yaml:"port"`
	Timestamps bool                    `yaml:"timestamps"`
	WindowSize int                     `yaml:"windowsize"`
}

// type DatabaseConfig struct {
// 	Metrics []MetricConfig `yaml:"metrics"`
// }

type MetricConfig struct {
	Aggregations map[string]Aggregation `yaml:"aggregations"`
}

func toInt(i interface{}) int {
	switch v := i.(type) {
	case uint64:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case nil:
		return 0
	default:
		panic(fmt.Sprintf("Cant convert %v of type %T to int", v, v))
	}
}

func toBool(i interface{}) bool {
	switch v := i.(type) {
	case bool:
		return v
	case nil:
		return false
	default:
		panic(fmt.Sprintf("Cant convert %v of type %T to bool", v, v))
	}
}

func GetLogLevel(logLevelString string) slog.Level {
	switch logLevelString {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type BucketParams struct {
	Start       int
	Width       int
	Count       int
	NoneCounter bool
	UseMidpoint bool
}

func (mC *MetricConfig) IsBucket(label string) (bool, BucketParams) {
	aggregation, ok := mC.Aggregations[label]
	if !ok || !strings.EqualFold(aggregation.Type, "Bucket") {
		return false, BucketParams{}
	}
	return true,
		BucketParams{
			Start:       toInt(aggregation.Params["start"]),
			Width:       toInt(aggregation.Params["width"]),
			Count:       toInt(aggregation.Params["count"]),
			NoneCounter: toBool(aggregation.Params["none_counter"]),
			UseMidpoint: toBool(aggregation.Params["use_midpoint"]),
		}
}

func (mC *MetricConfig) IsEliminateDimension(label string) bool {
	aggregation, ok := mC.Aggregations[label]
	if !ok || !strings.EqualFold(aggregation.Type, "EliminateDimension") {
		return false
	}
	return true
}

type MaxCellsParams struct {
	X int
}

func (mC *MetricConfig) IsMaxCells(label string) (bool, MaxCellsParams) {
	aggregation, ok := mC.Aggregations[label]
	if !ok || !strings.EqualFold(aggregation.Type, "MaxCells") {
		return false, MaxCellsParams{}
	}
	return true, MaxCellsParams{toInt(aggregation.Params["x"])}
}

func (mC *MetricConfig) IsFilter(label string) (bool, []string) {
	aggregation, ok := mC.Aggregations[label]
	if !ok || !strings.EqualFold(aggregation.Type, "Filter") {
		return false, []string{}
	}
	allowedValues := []string{}
	for key := range aggregation.Params {
		allowedValues = append(allowedValues, key)
	}
	return true, allowedValues
}

type Aggregation struct {
	Type   string                 `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ParseConfigText(content []byte) Config {
	var config Config

	//Set defaults
	config.RemoveReadFiles = DefaultRemoveReadFiles
	config.Interval = DefaultInterval
	config.DataDir = DefaultDataDir
	config.LogLevel = DefaultLogLevel
	config.Prometheus = PrometheusConfig{Port: DefaultPrometheusPort, Timestamps: DefaultTimestamps, WindowSize: DefaultWindowSize}

	err := yaml.Unmarshal(content, &config)
	checkError(err)

	return config
}

func ParseConfig(path string) Config {
	fileContent, err := os.ReadFile(path)
	checkError(err)

	return ParseConfigText(fileContent)
}
