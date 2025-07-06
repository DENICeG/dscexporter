package main

import (
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
	assert.Equal(t, config.DefaultDataDir, conf.DataDir)
}

func TestAllParams(t *testing.T) {
	args := []string{"--config=./testdata/config.yaml", "--data=./testdata/dsc-data", "--interval=30s", "--no-remove", "--port", "2114"}
	conf := ParamsToConfig(args)
	assert.Equal(t, 30*time.Second, conf.Interval)
	assert.Equal(t, 2114, conf.Prometheus.Port)
	assert.Equal(t, false, conf.RemoveReadFiles)
	assert.Equal(t, "./testdata/dsc-data", conf.DataDir)
}

func TestAllParamsShort(t *testing.T) {
	args := []string{"-c", "./testdata/config.yaml", "-d", "./testdata/dsc-data", "-i", "30s", "-p", "2114"}
	conf := ParamsToConfig(args)
	assert.Equal(t, 30*time.Second, conf.Interval)
	assert.Equal(t, 2114, conf.Prometheus.Port)
	assert.Equal(t, true, conf.RemoveReadFiles)
	assert.Equal(t, "./testdata/dsc-data", conf.DataDir)
}
