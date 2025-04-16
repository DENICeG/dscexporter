package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

const DefaultInterval = 15 * time.Second
const DefaultDataDir = "/data/exporter_dsc"
const DefaultRemoveReadFiles = false
const DefaultPrometheusPort = 2112

type Config struct {
	RemoveReadFiles bool             `yaml:"remove"`
	DataDir         string           `yaml:"data"`
	Interval        time.Duration    `yaml:"interval"`
	Prometheus      PrometheusConfig `yaml:"prometheus"`
	//Database DatabaseConfig `yaml:"database"`
}

type PrometheusConfig struct {
	Metrics map[string]MetricConfig `yaml:"metrics"`
	Port    int                     `yaml:"port"`
}

// type DatabaseConfig struct {
// 	Metrics []MetricConfig `yaml:"metrics"`
// }

type MetricConfig struct {
	Aggregations map[string]Aggregation `yaml:"aggregations"`
}

type BucketParams struct {
	Start int
	Width int
	Count int
}

func (mC *MetricConfig) IsBucket(label string) (bool, BucketParams) {
	aggregation, ok := mC.Aggregations[label]
	if !ok {
		return false, BucketParams{}
	}
	return strings.EqualFold(aggregation.Type, "Bucket"),
		BucketParams{
			Start: aggregation.Params["start"],
			Width: aggregation.Params["width"],
			Count: aggregation.Params["count"],
		}
}

func (mC *MetricConfig) IsEliminateDimension(label string) bool {
	aggregation, ok := mC.Aggregations[label]
	if !ok {
		return false
	}
	return strings.EqualFold(aggregation.Type, "EliminateDimension")
}

type MaxCellsParams struct {
	X int
}

func (mC *MetricConfig) IsMaxCells(label string) (bool, MaxCellsParams) {
	aggregation, ok := mC.Aggregations[label]
	if !ok {
		return false, MaxCellsParams{}
	}
	return strings.EqualFold(aggregation.Type, "MaxCells"), MaxCellsParams{aggregation.Params["x"]}
}

type Aggregation struct {
	Type   string         `yaml:"type"`
	Params map[string]int `yaml:"params"`
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
	config.Prometheus = PrometheusConfig{Port: DefaultPrometheusPort}

	err := yaml.Unmarshal(content, &config)
	checkError(err)

	return config
}

func ParseConfig(path string) Config {
	fileContent, err := os.ReadFile(path)
	checkError(err)

	return ParseConfigText(fileContent)
}
