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
	registryId = kingpin.Flag("repository", "Aws ecr repository id. Uses default when omitted.").Envar("ESU_ECR_REGISTRY_ID").Default("").String()
	allRepos   = kingpin.Flag("check-all", "Get scan results for all repositories in the registry").Default("false").Bool()
	latestTag  = kingpin.Flag("latest", "Get result for most recent image for specified repo. Ignores version of supplied composition if present.").Default("false").Bool()
	// The following Options are used together to parse a composition file in yaml format.
	composition   = kingpin.Flag("composition", "ZD Composition file to load when running batch mode.").Envar("ESU_COMPOSITION_FILE").Default("").String()
	whitelistFile = kingpin.Flag("whitelist", "Whitelist file containing package substrings to ignore per image and/or globally").Envar("ESU_WHITELIST_FILE").Default("").String()
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
	//Parse arguments
	kingpin.Parse()

	//Setup shared logger
	L := logger.Init("ESU Logger", *verbose, false, ioutil.Discard)
	logger.SetFlags(log.LUTC)

	//Load and create optional whitelist
	whitelist, err := helpers.CreateWhitelist(*whitelistFile, *L)
	helpers.Check(err, *L, "Failed to return whitelist.")

	//Configuring and creating shared session
	awsConfig := helpers.NewDefaultAwsConfig()
	s, sErr := session.NewSession(&awsConfig)
	helpers.Check(sErr, *L, "Failed to create session.")

	//Decision tree
	if *composition != "" {
		config := helpers.NewDefaultCompositionConfig(composition, baseRepo, stripPrefix, stripSuffix)
		L.Info("Reading Composition...")
		cl, err := helpers.CompositionParser(&config, *L)
		helpers.Check(err, *L, "Failed to generate container list")
		for i := range cl {
			_ = doSingleReport(&cl[i], s, &whitelist, *L)
		}

	} else if *allRepos {
		//Grab all repo's
		allRepositories, err := helpers.GetEcrRepositories(registryId, s, *L)
		helpers.Check(err, *L)
		for r := range allRepositories {
			//Create
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
				},
				s, *L)
			helpers.Check(err, *L)

			_ = doSingleReport(&image, s, &whitelist, *L)

		}

	} else {
		repositoryName := strings.Join([]string{*baseRepo, *containerName}, "/")

		image := helpers.NewImageDefinition(repositoryName, *containerTag)

		if *latestTag || *containerTag == "" {
			image.ImageId.ImageTag, _ = helpers.GetLatestTag(&ecr.Repository{
				RegistryId:     image.RegistryId,
				RepositoryName: image.RepositoryName,
			}, s, *L)
		} else {
			image.ImageId.ImageTag = aws.String(*containerTag)
		}
		_ = doSingleReport(&image, s, &whitelist, *L)
	}
}

func doSingleReport(image *ecr.Image, session *session.Session, whitelist *helpers.Whitelist, l logger.Logger) error {
	reporterConfig := helpers.NewCustomReporterConfig(helpers.FileNameFormatter(*image.RepositoryName), fmt.Sprintf("%s/", *reportDir), *reporterList)
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

			re := reporters.CreateXmlReport(*image.RepositoryName, *severityCutoff, *result.ImageScanFindings, reporterConfig, &componentWhitelist, l)
			helpers.Check(re, l, "Failed to write report for %s", *containerName)
		}
	} else {
		l.Warningf("Scan failed for %s: %v", n, *result.ImageScanStatus.Description)
	}
	return err
}
