package main

import (
	"log/slog"
	"testing"
	"time"

	"github.com/DENICeG/dscexporter/config"
	"github.com/stretchr/testify/assert"
)

func TestOnlyConfig(t *testing.T) {
	args := []string{"--config=./testdata/config.yaml"}
	conf := ParamsToConfig(args)
	assert.Equal(t, 20*time.Second, conf.Interval)
	assert.Equal(t, 2113, conf.Prometheus.Port)
	assert.Equal(t, true, conf.RemoveReadFiles)
	assert.Equal(t, slog.LevelDebug, conf.LogLevel)
	assert.Equal(t, config.DefaultDataDir, conf.DataDir)
	assert.Equal(t, false, conf.Prometheus.Timestamps)
	assert.Equal(t, 3, conf.Prometheus.WindowSize)
}

func TestAllParams(t *testing.T) {
	args := []string{"--config=./testdata/config.yaml", "--data=./testdata/dsc-data", "--interval=30s", "--no-remove", "--port", "2114", "--log-level", "debug", "--no-timestamps", "--windowsize", "10"}
	conf := ParamsToConfig(args)
	assert.Equal(t, 30*time.Second, conf.Interval)
	assert.Equal(t, 2114, conf.Prometheus.Port)
	assert.Equal(t, false, conf.RemoveReadFiles)
	assert.Equal(t, slog.LevelDebug, conf.LogLevel)
	assert.Equal(t, "./testdata/dsc-data", conf.DataDir)
	assert.Equal(t, false, conf.Prometheus.Timestamps)
	assert.Equal(t, 10, conf.Prometheus.WindowSize)
}

func TestAllParamsShort(t *testing.T) {
	args := []string{"-c", "./testdata/config.yaml", "-d", "./testdata/dsc-data", "-i", "30s", "-p", "2114"}
	conf := ParamsToConfig(args)
	assert.Equal(t, 30*time.Second, conf.Interval)
	assert.Equal(t, 2114, conf.Prometheus.Port)
	assert.Equal(t, true, conf.RemoveReadFiles)
	assert.Equal(t, "./testdata/dsc-data", conf.DataDir)
}
