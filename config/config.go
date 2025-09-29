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

func (p *PrometheusConfig) IsInTimeWindow(timestamp time.Time) bool {
	endOfWindow := time.Now().Add(-time.Duration(p.WindowSize) * time.Minute)
	return timestamp.After(endOfWindow)
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
	Start int
	Width int
	Count int
}

func (b *BucketParams) Buckets() []float64 {
	buckets := make([]float64, 0)
	for i := 0.0; i < float64(b.Count); i++ {
		buckets = append(buckets, float64(b.Start)+float64(b.Width)*i)
	}
	return buckets
}

func (mC *MetricConfig) IsBucket(label string) (bool, BucketParams) {
	aggregation, ok := mC.Aggregations[label]
	if !ok || !strings.EqualFold(aggregation.Type, "Bucket") {
		return false, BucketParams{}
	}
	return true,
		BucketParams{
			Start: toInt(aggregation.Params["start"]),
			Width: toInt(aggregation.Params["width"]),
			Count: toInt(aggregation.Params["count"]),
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

func CheckError(err error) {
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
	CheckError(err)

	return config
}

func ParseConfig(path string) Config {
	fileContent, err := os.ReadFile(path)
	CheckError(err)

	return ParseConfigText(fileContent)
}

func GetLastFullMin() time.Time {
	nowUnix := time.Now().Unix()
	return time.Unix(nowUnix-nowUnix%60, 0)
}
