package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
	"gopkg.in/yaml.v2"
)

type ReporterConfig struct {
	ReportFileName string
	ReporterType   string
	ReportBaseDir  string
}

type CompositionConfig struct {
	CompositionFileName string
	BaseRepo            string
	StripPrefix         string
	StripSuffix         string
}

func NewDefaultCompositionConfig(compositionFile *string, baseRepo *string, stripPrefix *string, stripSuffix *string) CompositionConfig {
	return CompositionConfig{
		CompositionFileName: *compositionFile,
		BaseRepo:            *baseRepo,
		StripPrefix:         *stripPrefix,
		StripSuffix:         *stripSuffix,
	}
}
func NewImageDefinition(registryID *string, repositoryName string, imageTag string) (image ecr.Image) {
	return ecr.Image{
		ImageId: &ecr.ImageIdentifier{
			ImageTag: &imageTag,
		},
		RegistryId:     registryID,
		RepositoryName: aws.String(repositoryName),
	}
}

func NewCustomReporterConfig(filename string, basedir string, reporterType string) (config ReporterConfig) {
	return ReporterConfig{
		ReportFileName: filename,
		ReportBaseDir:  basedir,
		ReporterType:   reporterType,
	}
}

func NewDefaultAwsConfig(region *string) aws.Config {

	return aws.Config{
		Region: region,
	}
}

type GlobalConfig struct {
	AwsConfig      *aws.Config
	ReporterConfig ReporterConfig
}

func Check(e error, logger *logger.Logger, a ...interface{}) {
	if e != nil {
		logger.Error(a)
		panic(e)
	}
}

func CheckAndExit(e error, logger *logger.Logger, a ...interface{}) {
	if e != nil {
		logger.Fatal(a)
		os.Exit(1)
	}
}
func CompositionParser(s *CompositionConfig, r *string, l *logger.Logger) ([]ecr.Image, error) {
	// This takes the configuration file (as passed via struct) and returns a list of generic
	// container objects that can be used as input when interacting with the ECR endpoints.
	// Currently only uses tag identifiers. TODO: Abstract further to allow working with hashes.

	zdComposition := make(map[string]string)
	imageList := make([]ecr.Image, 0)

	yamlFile, err := fileReader(s.CompositionFileName, l)
	Check(err, l, "Failed to read file %s: %s", s.CompositionFileName, err)

	l.Infof("unmarshalling contents of %s", s.CompositionFileName)
	err = yaml.Unmarshal(yamlFile, zdComposition)
	Check(err, l, "Failed to unmarshal %s, %v", yamlFile, err)

	for c, v := range zdComposition {
		c = underscoreHyphenator(suffixStripper(prefixStripper(c, s.StripPrefix), s.StripSuffix))
		image := NewImageDefinition(r, strings.Join([]string{s.BaseRepo, c}, "/"), v)
		imageList = append(imageList, image)
	}

	return imageList, err
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

func FileNameFormatter(filename string) string {

	return path.Base(fmt.Sprintf("%s-%s.xml", filename, timeStamper()))
}

// StringPointerChecker guards against nil pointer issues and returns a message in case pointer is nil to avoid issues
// with optional fields.
func StringPointerChecker(pointer *string, message string) string {
	if pointer == nil {
		return message
	} else {
		return *pointer
	}
}

func timeStamper() string {
	t := time.Now().Format(time.RFC3339)
	t = strings.Replace(t, "Z", "", 1)
	t = strings.Replace(t, "-", "", -1)
	t = strings.Replace(t, ":", "", -1)
	t = strings.Replace(t, "T", "-", 1)
	return strings.Replace(t, " ", "", -1)
}

func suffixStripper(input string, suffix string) (output string) {

	//We have to do some extra work because some containers we run
	//use version in the middle of their name which we can't strip.

	i := strings.LastIndex(input, suffix)
	if i != -1 {
		return input[:i] + strings.Replace(input[i:], suffix, "", 1)
	} else {
		return input
	}
}

func prefixStripper(input string, prefix string) (output string) {

	//prefix stripper doesn't just removes the first instance

	return strings.Replace(input, prefix, "", 1)
}

func underscoreHyphenator(input string) (output string) {
	return strings.Replace(input, "_", "-", -1)
}

func fileReader(filename string, l *logger.Logger) ([]byte, error) {
	l.Infof("trying to read contents of %s", filename)
	return ioutil.ReadFile(filename)
}
