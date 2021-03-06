package reporters

import (
	"github.com/aws/aws-sdk-go/service/ecr"
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
