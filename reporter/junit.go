package reporters

import (
	"encoding/xml"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/onsi/ginkgo/reporters"
)

func NewTestSuite(component string, findings ecr.ImageScanFindings) (testSuite reporters.JUnitTestSuite, err error) {
	var failures int = int(*findings.FindingSeverityCounts["CRITICAL"])
	testSuite = reporters.JUnitTestSuite{
		XMLName:   xml.Name{component, nil},
		TestCases: nil,
		Name:      component,
		Tests:     len(findings.Findings),
		Failures:  failures,
		Errors:    0,
		Time:      0,
	}
	for f := range findings.Findings {
		finding := findings.Findings[f]

		testCase := reporters.JUnitTestCase{
			Name:           *finding.Name,
			ClassName:      component,
			PassedMessage:  nil,
			FailureMessage: nil,
			Skipped:        nil,
			Time:           0,
			SystemOut:      "",
		}
		testSuite.TestCases[f] = testCase
	}

	return testSuite, err
}
