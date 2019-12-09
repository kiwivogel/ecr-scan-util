package reporters

import (
	"encoding/xml"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ecr"
)

type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	TestCases []JUnitTestCase `xml:"testcase"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
}

type JUnitTestCase struct {
	Name           string               `xml:"name,attr"`
	ClassName      string               `xml:"classname,attr"`
	PassedMessage  *JUnitPassedMessage  `xml:"passed,omitempty"`
	FailureMessage *JUnitFailureMessage `xml:"failure,omitempty"`
	Skipped        *JUnitSkipped        `xml:"skipped,omitempty"`
	Time           float64              `xml:"time,attr"`
	SystemOut      string               `xml:"system-out,omitempty"`
}

type JUnitPassedMessage struct {
	Message string `xml:",chardata"`
}

type JUnitFailureMessage struct {
	Type    string `xml:"type,attr"`
	Message string `xml:",chardata"`
}

type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

func NewTestSuite(container string, cutoff string, findings ecr.ImageScanFindings) (testSuite JUnitTestSuite, err error) {
	failures, err := countFailures(cutoff, findings.FindingSeverityCounts)
	if err != nil {
		panic(err)
	}

	testSuite = JUnitTestSuite{
		XMLName:   xml.Name{container, "bla"},
		TestCases: nil,
		Name:      container,
		Tests:     len(findings.Findings),
		Failures:  failures,
		Errors:    int(getSeverityCount("UNDEFINED", findings.FindingSeverityCounts)),
		Time:      0,
	}
	for f := range findings.Findings {
		testSuite.TestCases = append(testSuite.TestCases, createTestCase(cutoff, *findings.Findings[f]))
	}

	return testSuite, err
}

func countFailures(cutoff string, severityCounts map[string]*int64) (failures int, err error) {
	var f int64 = 0
	switch cutoff {
	case "LOW":
		f = getSeverityCount("LOW", severityCounts)
		fallthrough
	case "MEDIUM":
		f = f + getSeverityCount("MEDIUM", severityCounts)
		fallthrough
	case "HIGH":
		f = f + getSeverityCount("HIGH", severityCounts)
		fallthrough
	case "CRITICAL":
		f = f + getSeverityCount("CRITICAL", severityCounts)
	}
	return int(f), err
}
func getSeverityCount(index string, severityCounts map[string]*int64) (count int64) {
	value, present := severityCounts[index]
	if present {
		return *value
	} else {
		return 0
	}
}

func hasPassedCutoff(cutoff string, severity string) (passed bool) {
	severityMap := map[string]int{
		"LOW":      0,
		"MEDIUM":   1,
		"HIGH":     2,
		"CRITICAL": 3,
	}
	if severityMap[severity] >= severityMap[cutoff] {
		return false
	} else {
		return true
	}

}

func createTestCase(cutoff string, finding ecr.ImageScanFinding) (testCase JUnitTestCase) {
	passed := hasPassedCutoff(cutoff, *finding.Severity)
	packageName, err := ExtractPackageAttributes("package_name", &finding)
	packageVersion, err := ExtractPackageAttributes("package_version", &finding)
	packageString := fmt.Sprintf("%s@%s", packageName, packageVersion)
	if err != nil {
		panic(err)
	}
	if passed {
		return JUnitTestCase{
			Name:           *finding.Name,
			ClassName:      packageString,
			PassedMessage:  newPassedMessage(*finding.Name, *finding.Severity, cutoff),
			FailureMessage: nil,
			Skipped:        nil,
			Time:           0,
			SystemOut:      "",
		}
	} else {
		return JUnitTestCase{
			Name:           *finding.Name,
			ClassName:      packageString,
			PassedMessage:  nil,
			FailureMessage: newFailedMessage(*finding.Name, *finding.Severity, cutoff, *finding.Description),
			Skipped:        nil,
			Time:           0,
			SystemOut:      "",
		}
	}
}

func newPassedMessage(name string, severity string, cutoff string) *JUnitPassedMessage {
	return &JUnitPassedMessage{
		Message: fmt.Sprintf("Vulnerability %s with severity %s below cutoff %s. PASSED!", name, severity, cutoff),
	}
}
func newFailedMessage(name string, severity string, cutoff string, description string) *JUnitFailureMessage {
	return &JUnitFailureMessage{
		Type:    severity,
		Message: fmt.Sprintf("Vulnerability %s of severity %s above cutoff %s. FAILED! Description: %s", name, severity, cutoff, description),
	}
}
