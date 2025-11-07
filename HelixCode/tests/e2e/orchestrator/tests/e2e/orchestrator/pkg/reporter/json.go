package reporter

import (
	"encoding/json"

	"dev.helix.code/tests/e2e/orchestrator/pkg"
)

// JSONReporter generates JSON format reports
type JSONReporter struct{}

// Generate creates a JSON report
func (r *JSONReporter) Generate(report *pkg.TestReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
