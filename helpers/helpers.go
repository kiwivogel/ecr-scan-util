package helpers

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type ReporterConfig struct {
	ReportFileName string
	ReporterType   string
	ReportBaseDir  string
}

func NewDefaultReporterConfig() (config ReporterConfig) {
	return ReporterConfig{
		ReportFileName: "testreport.xml",
		ReporterType:   "junit",
		ReportBaseDir:  "",
	}
}
func NewCustomReporterConfig(filename string, basedir string, reporterType string) (config ReporterConfig) {
	return ReporterConfig{
		ReportFileName: filename,
		ReportBaseDir:  basedir,
		ReporterType:   reporterType,
	}
}

type GlobalConfig struct {
	AwsConfig      *aws.Config
	ReporterConfig ReporterConfig
}

func Check(e error, message string) {
	if e != nil {
		fmt.Printf("%s \n %e", message, e)
		panic(e)
	}
}
func CompositionParser(compositionFile string) (map[string]string, error) {
	zdComposition := make(map[string]string)
	containerList := make(map[string]string)
	yamlFile, err := ioutil.ReadFile(compositionFile)
	Check(err, fmt.Sprintf("Failed to load %s, #%e", compositionFile, err))

	err = yaml.Unmarshal(yamlFile, zdComposition)
	Check(err, fmt.Sprintf("Failed to unmarshal %v, #%e", yamlFile, err))

	for c, v := range zdComposition {
		c = underscoreHyphenator(versionStripper(c))
		containerList[c] = v
	}

	return containerList, err
}

func ExtractPackageAttributes(query string, finding *ecr.ImageScanFinding) (attribute string, err error) {
	for a := range finding.Attributes {
		if *finding.Attributes[a].Key == query {
			attribute = *finding.Attributes[a].Value
		}
	}
	if attribute != "" {
		return attribute, nil
	} else {
		fmt.Printf("Query for key %s returned no hits or an emtpy value", query)
		return "", errors.New("query for returned an empty result or key has no associated value")
	}
}

func versionStripper(input string) (output string) {
	return strings.Replace(input, "_version", "", 1)
}

func underscoreHyphenator(input string) (output string) {
	return strings.Replace(input, "_", "-", -1)
}
