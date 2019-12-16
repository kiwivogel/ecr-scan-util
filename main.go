package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"github.com/kiwivogel/ecr-scan-util/helpers"
	"github.com/kiwivogel/ecr-scan-util/reporters"
	"gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

var (
	composition    = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESA_COMPOSITION_FILE").Default("").String()
	registryId     = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESA_ECR_REGISTRY_ID").Default("").String()
	baseRepo       = kingpin.Flag("baserepo", "Common prefix for images. E.g. zorgdomein").Envar("ESA_ECR_BASE_REPO").Default("zorgdomein").String()
	containerName  = kingpin.Flag("container", "Container name to fetch scan results for").Envar("ESA_ECR_CONTAINER_NAME").Default("nexus").String()
	containerTag   = kingpin.Flag("tag", "Container tag or hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_IDENTIFIER").Default("2.14.12-02-30102019").String()
	severityCutoff = kingpin.Flag("cutoff", "Severity to count as failures").Envar("ESA_SEVERITY_CUTOFF").Default("MEDIUM").String()
	//TODO: Implement hash based findings, Probably requires further abstraction of *ecrDescribeImageScanFindingsInput
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESA_ECR_CONTAINER_HASH").String()
	reporterList = kingpin.Flag("reporter", "Reporter(s) to use").Envar("ESA_REPORTERS").Default("junit").String()
	//TODO: make reporter config read a fucking yaml as option.
	//reporterConfigFile = kingpin.Flag("reporter", "Configuration file for configuring reporters").Envar("ESA_REPORTER_CONFIG").Default("").String()
)

///TODO: ADD MORE MESSAGES TO CHECK INVOCATIONS. This doesn't really help like this.

func main() {

	kingpin.Parse()

	findings := map[string]ecr.ImageScanFindings{}
	ReporterConfig := helpers.NewDefaultReporterConfig()

	if *composition != "" {

		cl, err := helpers.CompositionParser(*composition)
		helpers.Check(err, "failed to create containerlist")
		resultsArray, err := aggregator.BatchGetScanResultsByTag(cl, *registryId, *baseRepo)
		helpers.Check(err, "Failed to get results")
		for r := range resultsArray {
			findings[r] = *resultsArray[r].ImageScanFindings
		}

		for f := range findings {

			singleReporterConfig := helpers.NewCustomReporterConfig("report.xml", fmt.Sprintf("%s/", f), *reporterList)
			if singleReporterConfig.ReporterType == "junit" {
				se := reporters.CreateXmlReport(f, *severityCutoff, findings[f], singleReporterConfig)
				helpers.Check(se, fmt.Sprintf("Failed to write report for %s", f))
			}

		}

	} else {

		repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")

		result, err := aggregator.EcrGetScanResultsByTag(repositoryName, *containerTag, *registryId)
		helpers.Check(err, fmt.Sprintf("Failed to write report for %s:%s", *containerName, *containerTag))
		findings[*containerName] = *result.ImageScanFindings
		re := reporters.CreateXmlReport(repositoryName, *severityCutoff, findings[*containerName], ReporterConfig)
		helpers.Check(re, fmt.Sprintf("Failed to write report for %s", *containerName))
	}

}
