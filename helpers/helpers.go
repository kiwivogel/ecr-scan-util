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

// contains a bunch of helper functions / structs / concepts that we need across packages.

// ReporterConfig is a simple object we use to avoid parameter bloat
type ReporterConfig struct {
	ReportFileName string // ReportFileName: What filename to use when writing out the report file (if applicable), Liable to change when more reporters are added
	ReporterType   string // ReporterType: What reporter to use
	ReportBaseDir  string // ReportBaseDir: What directory to use when writing out the report file (if applicable), Liable to change when more reporters are added
}

// CompositionConfig is a simple object we use to avoid parameter bloat containing some parameters describing a CompositionFile and operations that need to be performed
// when converting it's contents to an []ecr.Image
// This tailored to ZorgDomein's usecase and may not suit your usecase.
type CompositionConfig struct {
	CompositionFileName string // CompositionFileName should contain a string with the path of a yaml file with a list of images to scan.
	BaseRepo            string // BaseRepo is a string that is added as <BaseRepo>/ImageName:ImageTag to avoid having to add shared prefices in CompositionFileName
	StripPrefix         string // StripPrefix is used to strip shared prefixes (such as "p_,rc_,r_") from the entries in CompositionFile when parsing that in CompositionParser
	StripSuffix         string // StripSuffix is used to strip shared suffixes (such as "_version") from the entries in CompositionFile when parsing that in CompositionParser
}

// NewCompositionConfig returns a CompositionConfig based on compositionFile (filepath to yaml file with composition, see README.MD for format), baserepo, stripPrefix, stripSuffix.
func NewCompositionConfig(compositionFile *string, baseRepo *string, stripPrefix *string, stripSuffix *string) CompositionConfig {
	return CompositionConfig{
		CompositionFileName: *compositionFile,
		BaseRepo:            *baseRepo,
		StripPrefix:         *stripPrefix,
		StripSuffix:         *stripSuffix,
	}
}

// NewImageDefinition takes a registryID (AWS account ID), repositoryName (name of container) and imageTag, returns a
// ecr.Image object.
func NewImageDefinition(registryID *string, repositoryName string, imageTag string) (image ecr.Image) {
	return ecr.Image{
		ImageId: &ecr.ImageIdentifier{
			ImageTag: &imageTag,
		},
		RegistryId:     registryID,
		RepositoryName: aws.String(repositoryName),
	}
}

// NewCustomReporterConfig Returns a ReporterConfig
func NewCustomReporterConfig(filename string, basedir string, reporterType string) (config ReporterConfig) {
	return ReporterConfig{
		ReportFileName: filename,
		ReportBaseDir:  basedir,
		ReporterType:   reporterType,
	}
}

// NewDefaultAwsConfig returns a default config with only region specified. Use profile config for other options.
func NewDefaultAwsConfig(region *string) aws.Config {

	return aws.Config{
		Region: region,
	}
}

// Check is a generic error check function we use for more readable code.
// take an error, *logger.Logger and an interface that we use to supply a template string and variables to populate that
// template string throws a panic.
func Check(e error, logger *logger.Logger, a ...interface{}) {
	if e != nil {
		logger.Error(a)
		panic(e)
	}
}

// CheckAndExit is a generic eror check function we use for more readable code.
// take an error, *logger.Logger and an interface that we use to supply a template string and variables to populate that
// template string we log as a Fatal then Exit 1's.
func CheckAndExit(e error, logger *logger.Logger, a ...interface{}) {
	if e != nil {
		logger.Fatal(a)
		os.Exit(1)
	}
}

// Composition parser is a a helper function to massage entries in a ZorgDomein flavoured composition file into a usable
// format. It takes a pointer to a CompositionConfig and returns a list of generic container objects that can be used as
// input when interacting with the ECR endpoints.
// Currently only uses tag identifiers.
// TODO: Abstract further to allow working with hashes.
// TODO: Make more generic for different use cases
func CompositionParser(s *CompositionConfig, r *string, l *logger.Logger) ([]ecr.Image, error) {

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

// ExtractPackageAttributes is a helper function used to query attributes in a given ecr.ImageScanFinding. We use this
// to guard against nil pointers/errors when a given attribute is not present. Returns the queried attribute if found or
// an error if the attribute is not present/nil.
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

// FileNameFormatter returns a timestamped filename string with an extension
func FileNameFormatter(filename string, fileExtension string) string {
	return path.Base(fmt.Sprintf("%s-%s.%s", filename, timeStamper(), fileExtension))
}

// StringPointerChecker guards against nil pointer issues and returns a message in case pointer is nil to avoid issues
// with optional fields. TODO: convert message to interface to allow for templatable messages.
func StringPointerChecker(pointer *string, message string) string {
	if pointer == nil {
		return message
	} else {
		return *pointer
	}
}

// timeStamper creates a timestamp in the format of YYYYMMDD-HHMMSS+ timezone offset of local timezone to create
// a more readable timestamp for the filename of report files
func timeStamper() string {
	// We use RFC3339 because that requires the least tinkering to get the format we want
	t := time.Now().Format(time.RFC3339)
	t = strings.Replace(t, "Z", "", 1)
	t = strings.Replace(t, "-", "", -1)
	t = strings.Replace(t, ":", "", -1)
	t = strings.Replace(t, "T", "-", 1)
	return strings.Replace(t, " ", "", -1)
}

// suffixStripper is used to optionally massage some common suffix you may or may not have in your compositionFile
// This is tailored to ZorgDomein's usecase and may not suit your usecase.
// returns an output string equal to the input minus last occurrence of the suffix supplied (or just the original string if suffix is not found)
func suffixStripper(input string, suffix string) (output string) {

	// We have to do some extra work because some containers may have the
	// input string in the middle of their name which we shouldn't strip.

	i := strings.LastIndex(input, suffix)
	if i != -1 {
		return input[:i] + strings.Replace(input[i:], suffix, "", 1)
	} else {
		return input
	}
}

// prefixStripper is used to optionalluy massage a common prefix you may or may not have in your compositionFile
// This tailored to ZorgDomein's usecase and may not suit your usecase.
// returns an output string equal to the input minus the prefix (or just the original string if suffix is not found)
func prefixStripper(input string, prefix string) (output string) {

	// This doet NOT check where the first occurance is.
	// TODO: Limit search space to string start.

	return strings.Replace(input, prefix, "", 1)
}

// underscoreHyphenator replaces all underscores by hyphens
func underscoreHyphenator(input string) (output string) {
	return strings.Replace(input, "_", "-", -1)
}

// fileReader tries to open a filename and returns a []byte and an error.
// also takes a logger.Logger to output what it's doing
func fileReader(filename string, l *logger.Logger) ([]byte, error) {
	l.Infof("trying to read contents of %s", filename)
	return ioutil.ReadFile(filename)
}
