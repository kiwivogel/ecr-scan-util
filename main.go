package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	reporters "github.com/kiwivogel/ecr-scan-util/reporter"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type ReporterConfig struct {
	reportFileName string
	reporterType   string
	reportBaseDir  string
}

func newDefaultReporterConfig() (config ReporterConfig) {
	return ReporterConfig{
		reportFileName: "testreport.xml",
		reporterType:   "junit",
		reportBaseDir:  "",
	}
}
func newCustomReporterConfig(filename string, basedir string, reporterType string) (config ReporterConfig) {
	return ReporterConfig{
		reportFileName: filename,
		reportBaseDir:  basedir,
		reporterType:   reporterType,
	}
}

type GlobalConfig struct {
	AwsConfig      *aws.Config
	ReporterConfig ReporterConfig
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	composition    = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESA_COMPOSITION_FILE").Default("").String()
	registryId     = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESA_ECR_REGISTRY_ID").Default("").String()
	baseRepo       = kingpin.Flag("baserepo", "Common prefix for images. E.g. zorgdomein").Envar("ESA_ECR_BASE_REPO").Default("zorgdomein").String()
	containerName  = kingpin.Flag("container", "Container name to fetch scan results for").Envar("ESA_ECR_CONTAINER_NAME").Default("nexus").String()
	containerTag   = kingpin.Flag("tag", "Container tag to fetch scan results for").Envar("ESA_ECR_CONTAINER_TAG").Default("2.14.12-02-30102019").String()
	severityCutoff = kingpin.Flag("cutoff", "Severity to count as failures").Envar("ESA_SEVERITY_CUTOFF").Default("MEDIUM").String()
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_HASH").String()
	reporterList = kingpin.Flag("reporter", "Reporter(s) to use").Envar("ESA_REPORTERS").Default("junit").String()
	//reporterConfigFile = kingpin.Flag("reporter", "Configuration file for configuring reporters").Envar("ESA_REPORTER_CONFIG").Default("").String()
)

func fileWriter(config ReporterConfig, input []byte) (err error) {
	var filepath = fmt.Sprintf("%s%s", config.reportBaseDir, config.reportFileName)

	file, err := os.Create(filepath)
	check(err)
	writer := bufio.NewWriter(file)
	ws, err := writer.WriteString(xml.Header)
	check(err)
	fmt.Printf("wrote header %d", ws)
	wf, err := writer.Write(input)
	check(err)
	fmt.Printf("writing results %d", wf)
	err = writer.Flush()
	check(err)
	return err
}

func main() {

	kingpin.Parse()

	repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")
	findings := map[string]ecr.ImageScanFindings{}
	ReporterConfig := newDefaultReporterConfig()

	if *composition != "" {
		yamlFile, err := ioutil.ReadFile(*composition)

		cl := make(map[string]string)

		if err != nil {
			log.Printf("Failed to load %s, #%v", *composition, err)
		}
		err = yaml.Unmarshal(yamlFile, cl)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		resultsArray, err := aggregator.BatchGetScanResultsByTag(cl, *registryId)
		for r := range resultsArray {
			findings[r] = *resultsArray[r].ImageScanFindings
		}

		for f := range findings {
			newCustomReporterConfig("report.xml", fmt.Sprintf("%s/", f), *reporterList)

		}

	} else {
		result, err := aggregator.EcrGetScanResultsByTag(repositoryName, *containerTag, *registryId)
		check(err)
		findings[*containerName] = *result.ImageScanFindings
		testSuite, err := reporters.NewTestSuite(*containerName, *severityCutoff, findings[*containerName])

		if err != nil {
			fmt.Printf("KAPOTSTUK %e", err)
		}
		bytes, err := xml.MarshalIndent(testSuite, "", "\t")
		if err != nil {
			fmt.Printf("KAPOTSTUK %e", err)
		}
		ioerr := fileWriter(ReporterConfig, bytes)
		check(ioerr)
	}

}
