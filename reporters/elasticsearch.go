package reporters

import (
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/elastic/go-elasticsearch"
	"github.com/google/logger"
	"github.com/kiwivogel/ecr-scan-util/helpers"
)

type VulnerabilityReport struct {
	Image          string
	ImageTag       string
	Name           string `json:"name"`
	Severity       string `json:"severity"`
	URI            string `json:"uri"`
	PackageName    string
	PackageVersion string
}

func CreateNewVulnerabilityReport(imageName string, imageTag string, finding *ecr.ImageScanFinding) (findingReport VulnerabilityReport, err error) {
	// if errors this is caused by an unexpected lack of package_name or package_version in a finding, this could technically
	// happen if something is wrong on AWS's side.
	packageName, err := helpers.ExtractPackageAttributes("package_name", finding)
	packageVersion, err := helpers.ExtractPackageAttributes("package_version", finding)
	if err != nil {
		return VulnerabilityReport{
			Image:          imageName,
			ImageTag:       imageTag,
			Name:           *finding.Name,
			Severity:       *finding.Severity,
			URI:            *finding.Uri,
			PackageName:    packageName,
			PackageVersion: packageVersion,
		}, err
	} else {
		return VulnerabilityReport{}, err
	}
}
func createEsClientConfig() elasticsearch.Config {
	return elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
			"http://lolcahost:9201",
		},
	}
}

func createEsClient(l *logger.Logger) (es *elasticsearch.Client, err error) {
	es, err = elasticsearch.NewClient(createEsClientConfig())
	if err != nil {
		return &elasticsearch.Client{}, err
	}
	return es, nil
}
