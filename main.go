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
	registryId = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESU_ECR_REGISTRY_ID").Default("").String()
	// The following Options are used together to parse a composition file in yaml format.
	composition   = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESU_COMPOSITION_FILE").Default("").ExistingFile()
	stripPrefix   = kingpin.Flag("strip-prefix", "Prefix string to strip while composition entries. Removes first occurrence of substring.").Default("").String()
	stripSuffix   = kingpin.Flag("strip-suffix", "Suffix string to strip while composition entries. Removes last occurrence of substring.").Default("_version").String()
	baseRepo      = kingpin.Flag("baserepo", "Prefix for images. will be prefixed onto entries in composition or containername supplied .").Envar("ESU_ECR_BASE_REPO").Default("").String()
	containerName = kingpin.Flag("container", "Container name to fetch scan results for").Envar("ESU_ECR_CONTAINER_NAME").Default("").String()
	containerTag  = kingpin.Flag("tag", "Container tag to fetch scan results for").Envar("ESU_ECR_CONTAINER_IDENTIFIER").Default("").String()
	//TODO: Implement hash based findings, Probably requires further abstraction of *ecrDescribeImageScanFindingsInput
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESU_ECR_CONTAINER_HASH").String()

	reportDir      = kingpin.Flag("directory", "Directory to write reports to").Envar("ESU_REPORT_DIR").Default("reports").String()
	severityCutoff = kingpin.Flag("cutoff", "Severity to count as failures").Envar("ESU_SEVERITY_CUTOFF").Default("MEDIUM").String()
	verbose        = kingpin.Flag("verbose", "log actions to stdout").Envar("ESU_VERBOSE_BOOL").Default("true").Bool()
	reporterList   = kingpin.Flag("reporter", "Reporter(s) to use").Envar("ESU_REPORTERS").Default("junit").String()
	//TODO: make reporter config read a fucking yaml as option.
	//reporterConfigFile = kingpin.Flag("reporter", "Configuration file for configuring reporters").Envar("ESU_REPORTER_CONFIG").Default("").String()
)

func main() {

	kingpin.Parse()

	L := logger.Init("ESU Logger", *verbose, false, ioutil.Discard)
	logger.SetFlags(log.LUTC)

	if *composition != "" {

		doCompositionBasedReports(*composition, *L)

	} else {

		doSingleReport(*L)

	}

}

func doCompositionBasedReports(composition string, l logger.Logger) {
	l.Info("Reading Composition...")
	cl, err := helpers.CompositionParser(composition, *stripPrefix, *stripSuffix, l)
	helpers.Check(err, l, "Failed to generate container list")

	l.Info("Getting Results for composition...")
	resultsArray, err := aggregator.BatchGetScanResultsByTag(cl, *registryId, *baseRepo, l)

	helpers.Check(err, l, "Failed to get results. \n")

	for r := range resultsArray {
		if *resultsArray[r].ImageScanStatus.Status != "FAILED" {
			results := *resultsArray[r].ImageScanFindings
			singleReporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(r), fmt.Sprintf("%s/", *reportDir), *reporterList)

			l.Info("Passing results to writer for ", r)
			if singleReporterConfig.ReporterType == "junit" {
				se := reporters.CreateXmlReport(r, *severityCutoff, results, singleReporterConfig, l)
				helpers.Check(se, l, "Failed to write report for %s", r)
			}
		} else {
			l.Warning("No results found for ", *resultsArray[r].RepositoryName, " Skipping")
		}
	}
}

func doSingleReport(l logger.Logger) {
	repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")
	reporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(*containerName), fmt.Sprintf("%s/", *reportDir), *reporterList)

	l.Info("Getting Results for container...")
	result, err := aggregator.EcrGetScanResultsByTag(repositoryName, *containerTag, *registryId, l)
	helpers.Check(err, l, "Failed to get results. \n")
	if *result.ImageScanStatus.Status != "FAILED" {

		l.Infof("Got results")
		if reporterConfig.ReporterType == "junit" {
			l.Infof("Creating junit test report")
			re := reporters.CreateXmlReport(repositoryName, *severityCutoff, *result.ImageScanFindings, reporterConfig, l)
			helpers.Check(re, l, "Failed to write report for %s", *containerName)
		}
	} else {
		l.Fatalf("Scan failed for %s: %v", repositoryName, result.ImageScanStatus.Description)
	}

}
