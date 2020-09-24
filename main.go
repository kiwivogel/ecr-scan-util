package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
	"github.com/kiwivogel/ecr-scan-util/aggregator"
	"github.com/kiwivogel/ecr-scan-util/helpers"
	"github.com/kiwivogel/ecr-scan-util/reporters"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "log actions to stdout").Envar("ESU_VERBOSE_BOOL").Default("true").Bool()
	//Generic settings used for setting up client
	registryId      = kingpin.Flag("registry-id", "Aws ecr repository id. Uses default when omitted.").Envar("ESU_ECR_REGISTRY_ID").Default("").String()
	baseRepo        = kingpin.Flag("base-repo", "Used when supplying image names with a common prefix").Envar("ESU_ECR_BASE_REPO").Default("").String()
	region          = kingpin.Flag("region", "AWS region").Default("").String()
	latestTag       = kingpin.Flag("latest-tag", "Get result for most recent tagged image for specified repo. Ignores version of supplied composition if present.").Default("false").Bool()
	latestTagFilter = kingpin.Flag("latest-tag-filter", "Ignores tags containing this substring.").Default("").String()

	reportCommand         = kingpin.Command("report", "Creates a report containing scan results from ECR's container scans")
	reportDir             = reportCommand.Flag("output-dir", "Directory to write reports to").Envar("ESU_REPORT_DIR").Default("reports").String()
	reportWhitelistFile   = reportCommand.Flag("whitelist", "Whitelist file containing package substrings to ignore per image and/or globally").Envar("ESU_WHITELIST_FILE").Default("").String()
	reportServerityCutoff = reportCommand.Flag("cutoff", "Severity to count as failures").Envar("ESU_SEVERITY_CUTOFF").Default("MEDIUM").String()
	reportReporters       = reportCommand.Flag("reporter", "Reporter(s) to use").Envar("ESU_REPORTERS").Default("junit").String()

	reportAllCommand = reportCommand.Command("all", "Iterate over all repositories in a given registry. (Finds latest tagged container and returns reports.)\n")

	reportSingleCommand       = reportCommand.Command("single", "Iterate over a single repository")
	reportSingleContainerName = reportSingleCommand.Flag("image-id", "Container name to fetch scan results for").Default("").String()
	reportSingleContainerTag  = reportSingleCommand.Flag("image-tag", "Container tag to fetch scan results for").Default("").String()

	reportCompositionCommand     = reportCommand.Command("composition", "Iterate over a user supplied list of Images (composition)")
	reportCompositionFile        = reportCompositionCommand.Flag("compositionfile", "ZD Composition file to load.").Default("").String()
	reportCompisotionStripPrefix = reportCompositionCommand.Flag("strip-prefix", "Prefix string to strip while parsing composition entries. Removes first occurrence of substring.").Default("").String()
	reportCompositionStripSuffix = reportCompositionCommand.Flag("strip-suffix", "Suffix string to strip while pasrsing composition entries. Removes last occurrence of substring.").Default("_version").String()

	//TODO: Implement hash based findings, Probably requires further abstraction of *ecrDescribeImageScanFindingsInput
	//containerHash =  kingpin.Flag("hash", "Container hash to fetch scan results for").Envar("ESU_ECR_CONTAINER_HASH").String()
	//TODO: make reporter config read a fucking yaml as option.
	//reporterConfigFile = reportCommand.Flag("reporter", "Configuration file for configuring reporters").Envar("ESU_REPORTER_CONFIG").Default("").String()
)

func main() {
	//Parse arguments
	kingpin.Parse()

	//Setup shared logger
	L := logger.Init("ESU Logger", *verbose, false, ioutil.Discard)
	logger.SetFlags(log.LUTC)

	//Load and create optional whitelist
	whitelist, err := helpers.CreateWhitelist(*reportWhitelistFile, *L)
	helpers.Check(err, L, "Failed to return whitelist.")

	//Configuring and creating shared session
	awsConfig := helpers.NewDefaultAwsConfig(region)
	s, sErr := session.NewSession(&awsConfig)
	helpers.Check(sErr, L, "Failed to create session.")
	//
	if *registryId == "" {
		registryId = nil
	}
	switch kingpin.Parse() {

	case reportAllCommand.FullCommand():
		err = doReportAll(&whitelist, s, L)
		helpers.CheckAndExit(err, L)

	case reportSingleCommand.FullCommand():
		err = doReportSingle(whitelist, s, L)
		helpers.CheckAndExit(err, L)

	case reportCompositionCommand.FullCommand():
		config := helpers.NewDefaultCompositionConfig(reportCompositionFile, baseRepo, reportCompisotionStripPrefix, reportCompositionStripSuffix)
		cl, err := helpers.CompositionParser(&config, registryId, L)
		helpers.CheckAndExit(err, L, "Failed to Parse file to extract list of images to iterate on")
		err = doReportComposition(cl, &whitelist, s, L)
		helpers.CheckAndExit(err, L)
	}
}

func doReportAll(w *helpers.Whitelist, s *session.Session, l *logger.Logger) error {
	//Grab all repo's
	allRepositories, err := helpers.GetEcrRepositories(registryId, s, *l)
	helpers.Check(err, l)
	for r := range allRepositories {
		image := ecr.Image{
			RepositoryName: allRepositories[r].RepositoryName,
			RegistryId:     registryId,
			ImageId: &ecr.ImageIdentifier{
				ImageTag: nil,
			},
		}
		image.ImageId.ImageTag, err = helpers.GetLatestTag(
			&ecr.Repository{
				RegistryId:     image.RegistryId,
				RepositoryName: image.RepositoryName,
			}, latestTagFilter, s, l)
		if err == nil {
			_ = createReport(&image, w, s, l)
		}
	}
	return nil

}

func doReportSingle(whitelist helpers.Whitelist, s *session.Session, l *logger.Logger) (err error) {
	image := helpers.NewImageDefinition(registryId, *reportSingleContainerName, *reportSingleContainerTag)
	if *latestTag == true {
		image.ImageId.ImageTag, err = helpers.GetLatestTag(&ecr.Repository{
			RegistryId:     image.RegistryId,
			RepositoryName: image.RepositoryName,
		}, latestTagFilter, s, l)
	}

	if *baseRepo != "" {
		image.RepositoryName = aws.String(strings.Join([]string{*baseRepo, *reportSingleContainerName}, "/"))

	}
	return createReport(&image, &whitelist, s, l)
}

func doReportComposition(images []ecr.Image, whitelist *helpers.Whitelist, session *session.Session, l *logger.Logger) error {
	for i := range images {
		if *latestTag {
			images[i].ImageId.ImageTag, _ = helpers.GetLatestTag(&ecr.Repository{
				RepositoryName: images[i].RepositoryName,
			}, latestTagFilter, session, l)
		}
		_ = createReport(&images[i], whitelist, session, l)

	}
	return nil
}

func createReport(image *ecr.Image, whitelist *helpers.Whitelist, session *session.Session, l *logger.Logger) error {
	reporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(*image.RepositoryName), fmt.Sprintf("%s/", *reportDir), *reportReporters)
	n := fmt.Sprintf("%s:%s", *image.RepositoryName, *image.ImageId.ImageTag)

	// Flatten global whitelist and component specific whitelist into a single array.
	// We convert repositoryName back into base name to keep whitelist readable
	componentWhitelist := helpers.FlattenWhitelist(whitelist, strings.TrimPrefix(*image.RepositoryName, fmt.Sprintf("%s/", *baseRepo)))

	l.Info("Getting Results for container: ", n)
	result, err := aggregator.EcrGetScanResults(image, session, l)
	if err != nil {
		return err
	} else if *result.ImageScanStatus.Status != "FAILED" {
		l.Infof("Got results")
		if reporterConfig.ReporterType == "junit" {
			l.Infof("Creating junit test report")

			re := reporters.CreateXmlReport(*image.RepositoryName, *reportServerityCutoff, *result.ImageScanFindings, reporterConfig, &componentWhitelist, l)
			helpers.Check(re, l, "Failed to write report for %s:%s", image.RepositoryName, image.ImageId.ImageTag)
		}
	} else {
		l.Warningf("Scan failed for %s: %v", n, *result.ImageScanStatus.Description)
	}
	return err
}
