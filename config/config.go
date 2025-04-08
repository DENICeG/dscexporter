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

func (mC *MetricConfig) IsBucket(label string) bool {
	aggregation, ok := mC.Aggregations[label]
	if !ok {
		return false
	}
	return strings.EqualFold(aggregation.Type, "Bucket")
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
