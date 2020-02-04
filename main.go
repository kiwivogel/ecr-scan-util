package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ecr"
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
	composition   = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESU_COMPOSITION_FILE").Default("").String()
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
		config := helpers.NewDefaultCompositionConfig(composition, baseRepo, stripPrefix, stripSuffix)
		doCompositionBasedReports(&config, *L)

	} else {
		repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")

		image := helpers.NewImageDefinition(repositoryName, *containerTag)

		_ = doSingleReport(image, *L)
	}

}

func doCompositionBasedReports(settings *helpers.CompositionConfig, l logger.Logger) {
	l.Info("Reading Composition...")
	cl, err := helpers.CompositionParser(settings, l)
	helpers.Check(err, l, "Failed to generate container list")

	for i := range cl {
		_ = doSingleReport(cl[i], l)
	}

}

func doSingleReport(image ecr.Image, l logger.Logger) error {
	reporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(*image.RepositoryName), fmt.Sprintf("%s/", *reportDir), *reporterList)
	n := fmt.Sprintf("%s:%s", *image.RepositoryName, *image.ImageId.ImageTag)

	l.Info("Getting Results for container: ", n)
	result, err := aggregator.EcrGetScanResults(image, l)
	if err != nil {
		return err
	} else if *result.ImageScanStatus.Status != "FAILED" {
		l.Infof("Got results")
		if reporterConfig.ReporterType == "junit" {
			l.Infof("Creating junit test report")

			re := reporters.CreateXmlReport(*image.RepositoryName, *severityCutoff, *result.ImageScanFindings, reporterConfig, l)
			helpers.Check(re, l, "Failed to write report for %s", *containerName)
		}
	} else {
		l.Warningf("Scan failed for %s: %v", n, *result.ImageScanStatus.Description)
	}
	return err
}
