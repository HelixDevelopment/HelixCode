package reporter

import (
	"encoding/xml"
	"fmt"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// JUnitTestSuite represents a JUnit XML test suite
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a JUnit XML test case
type JUnitTestCase struct {
	Name      string         `xml:"name,attr"`
	ClassName string         `xml:"classname,attr"`
	Time      float64        `xml:"time,attr"`
	Failure   *JUnitFailure  `xml:"failure,omitempty"`
	Skipped   *JUnitSkipped  `xml:"skipped,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test
type JUnitSkipped struct {
	Message string `xml:"message,attr"`
}

// JUnitReporter generates JUnit XML format reports
type JUnitReporter struct{}

// Generate creates a JUnit XML report
func (r *JUnitReporter) Generate(report *pkg.TestReport) ([]byte, error) {
	suite := JUnitTestSuite{
		Name:      report.SuiteName,
		Tests:     report.TotalTests,
		Failures:  report.Failed,
		Skipped:   report.Skipped,
		Time:      report.Duration.Seconds(),
		TestCases: make([]JUnitTestCase, 0, len(report.Results)),
	}

	for _, result := range report.Results {
		testCase := JUnitTestCase{
			Name:      result.TestName,
			ClassName: report.SuiteName,
			Time:      result.Duration.Seconds(),
		}

		switch result.Status {
		case pkg.StatusFailed, pkg.StatusTimedOut:
			testCase.Failure = &JUnitFailure{
				Message: result.ErrorMsg,
				Type:    string(result.Status),
				Content: result.Output,
			}
		case pkg.StatusSkipped:
			testCase.Skipped = &JUnitSkipped{
				Message: result.ErrorMsg,
			}
		}

		suite.TestCases = append(suite.TestCases, testCase)
	}

	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate JUnit XML: %w", err)
	}

	// Add XML header
	xmlHeader := []byte(xml.Header)
	return append(xmlHeader, data...), nil
}
