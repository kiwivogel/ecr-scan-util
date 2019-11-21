package main

import (
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
)

type componentList struct {
	Components map[string]string `yaml:Versions`
}

func main() {
	filename, _ := filepath.Abs("./composition.yml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Printf("Failed to load %s, #%v", filename, err)
	}
	err = yaml.Unmarshal(yamlFile, components)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	aggregator.BatchGetScanResults(components)
}
