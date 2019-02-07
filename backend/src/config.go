package main

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pdf/golifx/common"
)

//Config represents the configuration
type Config struct {
	Duration       int
	LightID        uint64
	UpdateInterval int
	Transitions    []struct {
		Starttime  int
		Endtime    int
		Startvalue float64
		Endvalue   float64
		Type       string
	}
	Initialcolor common.Color
}

//ReadConfig reads the config at the given path and returns a Config object
func ReadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	err = yaml.Unmarshal(b, &cfg)
	return &cfg, err
}
