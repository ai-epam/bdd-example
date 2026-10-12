package authgate_test

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"authgate/steps"
)

// TestFeatures runs every feature under ./features. Set TAGS (e.g. TAGS='~@wip')
// to filter scenarios. A JSON summary is written to ../bdd-report.json.
func TestFeatures(t *testing.T) {
	opts := godog.Options{
		Output:        colors.Colored(os.Stdout),
		Format:        "pretty",
		Paths:         []string{"features"},
		Tags:          os.Getenv("TAGS"),
		Strict:        true, // undefined steps fail instead of being ignored
		NoColors:      os.Getenv("NO_COLOR") != "",
		StopOnFailure: false,
	}
	status := godog.TestSuite{
		Name:                "authgate",
		ScenarioInitializer: steps.InitializeScenario,
		Options:             &opts,
	}.Run()
	if err := steps.Report.Write("../bdd-report.json"); err != nil {
		t.Errorf("writing report: %v", err)
	}
	if status != 0 {
		t.Fatal("BDD suite failed")
	}
}
