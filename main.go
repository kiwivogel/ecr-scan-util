package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

func main() {

	composition := false
	filename, _ := filepath.Abs("./composition.yml")

	baseRepo := "zorgdomein"
	componentName := "nexus"
	imageTag := "2.14.12-02-30102019"
	repositoryName := strings.Join([]string{baseRepo, componentName}, "/")

	if composition {
		yamlFile, err := ioutil.ReadFile(filename)

		cl := make(map[interface{}]interface{})

		if err != nil {
			log.Printf("Failed to load %s, #%v", filename, err)
		}
		err = yaml.Unmarshal(yamlFile, cl)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
		aggregator.BatchGetScanResults(cl)
	} else {

		findings := map[string]ecr.ImageScanFindings{}
		result, err := aggregator.EcrGetTagScanResults(repositoryName, imageTag)
		if err != nil {
			panic(err)
		}
		findings[componentName] = *result.ImageScanFindings

		fmt.Printf("DEBUG:: result for image %v: %v\n", componentName, findings[componentName].Findings)

	}

}
