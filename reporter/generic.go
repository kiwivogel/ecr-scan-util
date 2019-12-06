package reporters

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ecr"
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

func extractPackageAttributes(query string, finding *ecr.ImageScanFinding) (attribute string, err error) {
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

func CreateNewVulnerabilityReport(imageName string, imageTag string, finding *ecr.ImageScanFinding) (findingReport VulnerabilityReport, err error) {
	packageName, err := extractPackageAttributes("package_name", finding)
	packageVersion, err := extractPackageAttributes("package_version", finding)
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
