package auditcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAuditCheck(t *testing.T) {
	analyzer := &plugin{}
	analyzers, err := analyzer.BuildAnalyzers()
	if err != nil {
		t.Fatal(err)
	}

	analysistest.Run(t, analysistest.TestData(), analyzers[0], "example")
}
