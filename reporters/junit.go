package reporters

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/google/logger"
	"github.com/kiwivogel/ecr-scan-util/helpers"
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

func CreateXmlReport(container string, cutoff string, findings ecr.ImageScanFindings, config helpers.ReporterConfig, whitelist *[]string, l *logger.Logger) (err error) {

	s := newTestSuite(container, cutoff, findings, whitelist)
	we := xmlReportWriter(config, s, l)
	helpers.Check(we, l, "Failed to write file.\n")
	return err
}

func newTestSuite(container string, cutoff string, findings ecr.ImageScanFindings, whitelist *[]string) (testSuite JUnitTestSuite) {
	testSuite = JUnitTestSuite{
		XMLName:   xml.Name{container, "bla"},
		TestCases: nil,
		Name:      container,
		Tests:     len(findings.Findings),
		Failures:  countFailures(cutoff, findings.FindingSeverityCounts),
		Errors:    int(getSeverityCount("UNDEFINED", findings.FindingSeverityCounts)),
		Time:      0,
	}
	for f := range findings.Findings {
		testSuite.TestCases = append(testSuite.TestCases, createTestCase(cutoff, container, *findings.Findings[f], whitelist))
	}
	return testSuite
}

func countFailures(cutoff string, severityCounts map[string]*int64) (failures int) {
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
	return int(f)
}

func getSeverityCount(index string, severityCounts map[string]*int64) (count int64) {
	value, present := severityCounts[index]
	if present {
		return *value
	} else {
		return 0
	}
}

func hasPassedCutoff(cutoff string, severity string) bool {
	severityMap := map[string]int{
		"INFORMATIONAL": -1,
		"LOW":           0,
		"MEDIUM":        1,
		"HIGH":          2,
		"CRITICAL":      3,
	}
	return !(severityMap[severity] >= severityMap[cutoff])
}

func createTestCase(cutoff string, container string, finding ecr.ImageScanFinding, whitelist *[]string) (testCase JUnitTestCase) {
	passed := hasPassedCutoff(cutoff, *finding.Severity)

	packageName, err := helpers.ExtractPackageAttributes("package_name", &finding)
	packageVersion, err := helpers.ExtractPackageAttributes("package_version", &finding)

	packageString := fmt.Sprintf("%s@%s", packageName, packageVersion)

	allowListed, hit := helpers.InAllowList(*whitelist, packageString)

	if err != nil {
		panic(err)
	}
	testCase = JUnitTestCase{
		Name:      container,
		ClassName: packageString,
		Skipped:   nil,
		Time:      0,
		SystemOut: "",
	}
	if allowListed {
		testCase.PassedMessage = newGenericPassedMessage("Vulnerability %s with severity %s matches queried allowListed pattern %s. PASSED!",
			*finding.Name, *finding.Severity, hit)
		return testCase
	} else if passed {
		testCase.PassedMessage = newGenericPassedMessage("Vulnerability %s with severity %s below cutoff %s. PASSED!",
			*finding.Name, *finding.Severity, cutoff)
	} else {
		testCase.FailureMessage = newGenericFailedMessage(*finding.Severity,
			"Vulnerability %s of severity %s above cutoff %s. FAILED! Description: %s",
			*finding.Name, *finding.Severity, cutoff, helpers.StringPointerChecker(finding.Description, "No description provided"))
	}
	return testCase
}

func newGenericPassedMessage(template string, m ...interface{}) *JUnitPassedMessage {
	return &JUnitPassedMessage{
		Message: fmt.Sprintf(template, m...),
	}
}

func newGenericFailedMessage(severity string, template string, m ...interface{}) *JUnitFailureMessage {
	return &JUnitFailureMessage{
		Type:    severity,
		Message: fmt.Sprintf(template, m...),
	}
}

func xmlReportWriter(config helpers.ReporterConfig, suite JUnitTestSuite, l *logger.Logger) (err error) {

	var filepath = path.Join(config.ReportBaseDir, config.ReportFileName)

	if config.ReportBaseDir != "" {
		if _, err := os.Stat(config.ReportBaseDir); os.IsNotExist(err) {
			err := os.Mkdir(config.ReportBaseDir, 0744)
			helpers.Check(err, l, "Failed to create directory %s\n", config.ReportBaseDir)
		}
		helpers.Check(err, l, "")
	}

	formattedSuite, e := xml.MarshalIndent(suite, "", "\t")
	helpers.Check(e, l, "Failed to marshall and indent xml, %v\n", err)

	file, err := os.Create(filepath)
	defer closeFile(file, l)
	helpers.Check(err, l, "")
	writer := bufio.NewWriter(file)

	l.Infof("writing header tp %s", filepath)
	_, err = writer.WriteString(xml.Header)
	helpers.Check(err, l, "Failed to write header to %s: %v", filepath, err)

	l.Infof("writing results to %s", filepath)
	_, err = writer.Write(formattedSuite)
	helpers.Check(err, l, "Failed to write results to %s: %v", filepath, err)

	err = writer.Flush()
	return err
}

func closeFile(file *os.File, l *logger.Logger) {
	err := file.Close()
	helpers.CheckAndExit(err, l, "Failed to close file : %v", file, err)
}
