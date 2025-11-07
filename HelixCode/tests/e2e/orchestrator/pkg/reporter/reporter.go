package reporter

import (
	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// Reporter formats and outputs test reports
type Reporter interface {
	Generate(report *pkg.TestReport) ([]byte, error)
}

// Format represents the report output format
type Format string

const (
	FormatJSON  Format = "json"
	FormatJUnit Format = "junit"
)

// NewReporter creates a reporter for the specified format
func NewReporter(format Format) Reporter {
	switch format {
	case FormatJSON:
		return &JSONReporter{}
	case FormatJUnit:
		return &JUnitReporter{}
	default:
		return &JSONReporter{}
	}
}
