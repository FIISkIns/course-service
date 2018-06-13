package main

import "github.com/kelseyhightower/envconfig"

type ConfigurationSpec struct {
	Port int    `default:"7310"`
	Path string `default:"courses/example"`
}

var config ConfigurationSpec

func initConfig() {
	envconfig.MustProcess("course", &config)
}
