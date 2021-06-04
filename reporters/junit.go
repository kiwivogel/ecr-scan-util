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

// JUnit formatted testsuite which we abuse here as a container for individual findings (stored in this struct as JUnitTestCase)
// We do this because this can be easilly used to view scan results in Jenkins or similar.
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`     //XML header information
	TestCases []JUnitTestCase `xml:"testcase"`      //List of JUnitTestCase's with additional information on the spefic fidingin
	Name      string          `xml:"name,attr"`     //Name of container scanned
	Tests     int             `xml:"tests,attr"`    //Number of Ignored findings (counted as passed tests)
	Failures  int             `xml:"failures,attr"` //Number of Failures in findings
	Errors    int             `xml:"errors,attr"`   //Number or Errors in suite
	Time      float64         `xml:"time,attr"`     //Normally duration of test, no sense in using this here. Added to satisfy JUnit format.
}

// Individual JUnitTestCase which we use to store a single finding from an ECR container scan
type JUnitTestCase struct {
	Name           string               `xml:"name,attr"`            //Name of the container scanned
	ClassName      string               `xml:"classname,attr"`       //Used to store package name in this finding.
	PassedMessage  *JUnitPassedMessage  `xml:"passed,omitempty"`     //Message if finding passes Cutoff or Allowlist
	FailureMessage *JUnitFailureMessage `xml:"failure,omitempty"`    //Message if finding does not pass Cutoff
	Skipped        *JUnitSkipped        `xml:"skipped,omitempty"`    //Message if finding is counted as skipped. Unimplemented
	Time           float64              `xml:"time,attr"`            //Normally duration of test, no sense in using this here. Added to satisfy JUnit format.
	SystemOut      string               `xml:"system-out,omitempty"` //Normally used for stacktrace etc. of failed test. Added to satisfy JUnit format.
}

// Used to store message for "Passed" findings. Either because they are allowlisted of because the severity is below the cutoff
// TODO: consider splitting this across JUnitPassedMessage and JUnitSkipped (the latter for allowListed findings)
type JUnitPassedMessage struct {
	Message string `xml:",chardata"`
}

// Used to store Message and Severity for a finding that does not pass Cutoff of Allowlist
type JUnitFailureMessage struct {
	Type    string `xml:"type,attr"`
	Message string `xml:",chardata"`
}

// Currently unused, see JUnitPassedMessage's TODO.
type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

// CreateXmlReport takes a container(name), a cutoff parameter (either 'LOW', 'MEDIUM', 'HIGH' or 'CRITICAL') a list of findings of type ecr.ImageScanFindings,
// a helpers.ReporterConfig struct containing settings for file writeout and an allowList and writes out an XML JUnit report.
// returns an error upon failure.
func CreateXmlReport(container string, cutoff string, findings ecr.ImageScanFindings, config helpers.ReporterConfig, allowList *[]string, l *logger.Logger) (err error) {

	s := newTestSuite(container, cutoff, findings, allowList)
	we := xmlReportWriter(config, s, l)
	helpers.Check(we, l, "Failed to write file.\n")
	return err
}

// newTestSuite generates a populated JUnitTestSuite for a container (name) for a set of ecr.ImageScanFindings failing or passing individual
// cases based on a cutoff (either 'LOW', 'MEDIUM', 'HIGH' or 'CRITICAL') and an allowList. Returns a JUnitTestSuite.
func newTestSuite(container string, cutoff string, findings ecr.ImageScanFindings, allowList *[]string) (testSuite JUnitTestSuite) {
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
		testSuite.TestCases = append(testSuite.TestCases, createTestCase(cutoff, container, *findings.Findings[f], allowList))
	}
	return testSuite
}

// countFailures is used by newTestSuite to tally the amount of findings marked as `Failed`. Takes a cutoff (either 'LOW',
// 'MEDIUM', 'HIGH' or 'CRITICAL') and the FindingSeverityCounts map from ecr.ImageScanFindings. Returns a flat number based
// on cutoff parameter.
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

// getSeverityCount Extracts flat number from map given an index for tallying.
func getSeverityCount(index string, severityCounts map[string]*int64) (count int64) {
	value, present := severityCounts[index]
	if present {
		return *value
	} else {
		return 0
	}
}

// hasPassedCutoff is a helper function used by createTestCase to compare a findings severity to the cutoff to pass or fail a test
// returns false if counted as failed and true if passed.
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

// createTestCase converts a ecr.ImageScanFinding to an annotated JUnitTestCase
// takes a cutoff (either 'LOW', 'MEDIUM', 'HIGH' or 'CRITICAL') container (name), ecr.ImageScanFinding and an allowList
// TODO: Handle errors. (Not likely but possible if something is wrong with the data supplied by aws)
func createTestCase(cutoff string, container string, finding ecr.ImageScanFinding, allowList *[]string) (testCase JUnitTestCase) {
	passed := hasPassedCutoff(cutoff, *finding.Severity)

	packageName, err := helpers.ExtractPackageAttributes("package_name", &finding)
	packageVersion, err := helpers.ExtractPackageAttributes("package_version", &finding)

	packageString := fmt.Sprintf("%s@%s", packageName, packageVersion)

	allowListed, hit := helpers.InAllowList(*allowList, packageString)

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

//  newGenericPassedMessage takes a template string and an interface to return a formatted pointer to a JUnitPassedMessage
func newGenericPassedMessage(template string, m ...interface{}) *JUnitPassedMessage {
	return &JUnitPassedMessage{
		Message: fmt.Sprintf(template, m...),
	}
}

// newGenericFailedMessage takes a template string and an interface to return a formatted pointer to a JUnitFailureMessage
func newGenericFailedMessage(severity string, template string, m ...interface{}) *JUnitFailureMessage {
	return &JUnitFailureMessage{
		Type:    severity,
		Message: fmt.Sprintf(template, m...),
	}
}

// xmlReportWriter takes a helpers.ReporterConfig a JUnitTestSuite and a pointer to a logger.Logger	and writes an XML to
// disk (based on parameters suppplied in the ReporterConfig).
// Returns an error if this fails.

func xmlReportWriter(config helpers.ReporterConfig, suite JUnitTestSuite, l *logger.Logger) (err error) {

	var filepath = path.Join(config.ReportBaseDir, config.ReportFileName)

	// Attempt to handle non-existant ReportBaseDir by creating one if specified.
	if config.ReportBaseDir != "" {
		if _, err := os.Stat(config.ReportBaseDir); os.IsNotExist(err) {
			err := os.Mkdir(config.ReportBaseDir, 0744)
			helpers.Check(err, l, "Failed to create directory %s\n", config.ReportBaseDir)
		}
		helpers.Check(err, l, "")
	}

	// Massage JUnitTestSuite into nicely formatted xml
	formattedSuite, e := xml.MarshalIndent(suite, "", "\t")
	helpers.Check(e, l, "Failed to marshall and indent xml, %v\n", err)

	// Create and open file to write to (using defer to gracefully cleanup/close)
	file, err := os.Create(filepath)
	defer closeFile(file, l)
	helpers.Check(err, l, "")
	writer := bufio.NewWriter(file)

	// Write out XML header tp file
	l.Infof("writing header tp %s", filepath)
	_, err = writer.WriteString(xml.Header)
	helpers.Check(err, l, "Failed to write header to %s: %v", filepath, err)

	// Write out rest of suite to file
	l.Infof("writing results to %s", filepath)
	_, err = writer.Write(formattedSuite)
	helpers.Check(err, l, "Failed to write results to %s: %v", filepath, err)

	// Clear writer.
	err = writer.Flush()
	return err
}

// closeFile closes an open file (pointer to os.File) and logs (pointer to logger.Logger) upon failure.
func closeFile(file *os.File, l *logger.Logger) {
	err := file.Close()
	helpers.CheckAndExit(err, l, "Failed to close file : %v", file, err)
}
