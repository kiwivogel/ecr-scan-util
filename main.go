package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"github.com/kiwivogel/ecr-scan-util/helper"
	"github.com/kiwivogel/ecr-scan-util/reporter"
	"gopkg.in/alecthomas/kingpin.v2"
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

var (
	composition         = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESA_COMPOSITION_FILE").Default("").String()
	registryId          = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESA_ECR_REGISTRY_ID").Default("").String()
	baseRepo            = kingpin.Flag("baserepo", "Common prefix for images. E.g. zorgdomein").Envar("ESA_ECR_BASE_REPO").Default("zorgdomein").String()
	containerName       = kingpin.Flag("container", "Container name to fetch scan results for").Envar("ESA_ECR_CONTAINER_NAME").Default("nexus").String()
	containerIdentifier = kingpin.Flag("tag", "Container tag or hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_IDENTIFIER").Default("2.14.12-02-30102019").String()
	severityCutoff      = kingpin.Flag("cutoff", "Severity to count as failures").Envar("ESA_SEVERITY_CUTOFF").Default("MEDIUM").String()
	//TODO: Implement hash based findings, Probably requires further abstraction of *ecrDescribeImageScanFindingsInput
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_HASH").String()
	reporterList = kingpin.Flag("reporter", "Reporter(s) to use").Envar("ESA_REPORTERS").Default("junit").String()
	//TODO: make reporter config read a fucking yaml as option.
	//reporterConfigFile = kingpin.Flag("reporter", "Configuration file for configuring reporters").Envar("ESA_REPORTER_CONFIG").Default("").String()
)

///TODO: ADD MORE MESSAGES TO CHECK INVOCATIONS. This doesn't really help like this.

func fileWriter(config ReporterConfig, input []byte) (err error) {
	///TODO: Move to JUnit Or make more generic file output and put in helper.

	var filepath = fmt.Sprintf("%s%s", config.reportBaseDir, config.reportFileName)

	if config.reportBaseDir != "" {
		if _, err := os.Stat(config.reportBaseDir); os.IsNotExist(err) {
			err := os.Mkdir(config.reportBaseDir, 0744)
			helpers.Check(err, fmt.Sprintf("Failed to create directory %s", config.reportBaseDir))
		}
		helpers.Check(err, "")
	}

	file, err := os.Create(filepath)
	helpers.Check(err, "")
	writer := bufio.NewWriter(file)
	ws, err := writer.WriteString(xml.Header)
	helpers.Check(err, "")
	fmt.Printf("wrote header %d \n", ws)
	wf, err := writer.Write(input)
	helpers.Check(err, "")
	fmt.Printf("writing results %d \n", wf)
	err = writer.Flush()
	helpers.Check(err, "")
	return err
}

func main() {

	kingpin.Parse()

	findings := map[string]ecr.ImageScanFindings{}
	ReporterConfig := newDefaultReporterConfig()

	if *composition != "" {

		cl, err := helpers.CompositionParser(*composition)
		helpers.Check(err, "")
		resultsArray, err := aggregator.BatchGetScanResultsByTag(cl, *registryId, *baseRepo)
		for r := range resultsArray {
			findings[r] = *resultsArray[r].ImageScanFindings
		}

		for f := range findings {

			singleReporterConfig := newCustomReporterConfig("report.xml", fmt.Sprintf("%s/", f), *reporterList)
			singleTestSuite, err := reporters.NewTestSuite(f, *severityCutoff, findings[f])
			helpers.Check(err, "")
			singleBytes, err := xml.MarshalIndent(singleTestSuite, "", "\t")
			helpers.Check(err, "")
			singleIoerr := fileWriter(singleReporterConfig, singleBytes)
			helpers.Check(singleIoerr, "")
		}

	} else {

		repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")

		result, err := aggregator.EcrGetScanResultsByTag(repositoryName, *containerIdentifier, *registryId)
		helpers.Check(err, "")
		findings[*containerName] = *result.ImageScanFindings
		testSuite, err := reporters.NewTestSuite(*containerName, *severityCutoff, findings[*containerName])
		helpers.Check(err, "")
		bytes, err := xml.MarshalIndent(testSuite, "", "\t")
		if err != nil {
			fmt.Printf("KAPOTSTUK %e", err)
		}
		ioerr := fileWriter(ReporterConfig, bytes)
		helpers.Check(ioerr, "")
	}

}
