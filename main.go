package main

import (
	"fmt"
	"github.com/google/logger"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"github.com/kiwivogel/ecr-scan-util/helpers"
	"github.com/kiwivogel/ecr-scan-util/reporters"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"log"
	"strings"
)

var (
	composition    = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESA_COMPOSITION_FILE").Default("").String()
	registryId     = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESA_ECR_REGISTRY_ID").Default("").String()
	baseRepo       = kingpin.Flag("baserepo", "Common prefix for images. E.g. zorgdomein").Envar("ESA_ECR_BASE_REPO").Default("zorgdomein").String()
	containerName  = kingpin.Flag("container", "Container name to fetch scan results for").Envar("ESA_ECR_CONTAINER_NAME").Default("nexus").String()
	containerTag   = kingpin.Flag("tag", "Container tag or hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_IDENTIFIER").Default("2.14.12-02-30102019").String()
	reportDir      = kingpin.Flag("directory", "Directory to write reports to").Envar("ESA_REPORT_DIR").Default("reports").String()
	severityCutoff = kingpin.Flag("cutoff", "Severity to count as failures").Envar("ESA_SEVERITY_CUTOFF").Default("MEDIUM").String()
	verbose        = kingpin.Flag("verbose", "log actions to stdout").Envar("ESA_VERBOSE_BOOL").Default("true").Bool()
	//TODO: Implement hash based findings, Probably requires further abstraction of *ecrDescribeImageScanFindingsInput
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_HASH").String()
	reporterList = kingpin.Flag("reporter", "Reporter(s) to use").Envar("ESA_REPORTERS").Default("junit").String()
	//TODO: make reporter config read a fucking yaml as option.
	//reporterConfigFile = kingpin.Flag("reporter", "Configuration file for configuring reporters").Envar("ESA_REPORTER_CONFIG").Default("").String()
)

func main() {

	kingpin.Parse()

	L := logger.Init("ESA Logger", *verbose, false, ioutil.Discard)
	logger.SetFlags(log.LUTC)

	if *composition != "" {

		doCompositionBasedReports(*composition, *L)

	} else {

		doSingleReport(*L)

	}

}

func doCompositionBasedReports(composition string, l logger.Logger) {
	l.Info("Reading Composition...")
	cl, err := helpers.CompositionParser(composition, l)
	helpers.Check(err, l, "Failed to generate container list")

	l.Info("Getting Results for composition...")
	resultsArray, err := aggregator.BatchGetScanResultsByTag(cl, *registryId, *baseRepo, l)

	helpers.Check(err, l, "Failed to get results")

	for r := range resultsArray {
		results := *resultsArray[r].ImageScanFindings

		singleReporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(r), fmt.Sprintf("%s/", *reportDir), *reporterList)

		if singleReporterConfig.ReporterType == "junit" {
			se := reporters.CreateXmlReport(r, *severityCutoff, results, singleReporterConfig, l)
			helpers.Check(se, l, "Failed to write report for %s", r)
		}

	}
}

func doSingleReport(l logger.Logger) {

	repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")
	reporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(*containerName), fmt.Sprintf("%s/", *reportDir), *reporterList)

	l.Info("Getting Results for composition...")
	result, err := aggregator.EcrGetScanResultsByTag(repositoryName, *containerTag, *registryId, l)
	if err != nil && *result.ImageScanStatus.Status == "failed" {
		l.Fatal("No results found for ", repositoryName, " ", err)
	} else {
		l.Infof("Got results")
	}

	if reporterConfig.ReporterType == "junit" {
		l.Infof("Creating junit test report")
		re := reporters.CreateXmlReport(repositoryName, *severityCutoff, *result.ImageScanFindings, reporterConfig, l)
		helpers.Check(re, l, "Failed to write report for %s", *containerName)
	}
}
