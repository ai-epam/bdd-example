package steps

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
)

type ScenarioReport struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

type FeatureReport struct {
	Name      string           `json:"name"`
	Path      string           `json:"path"`
	Scenarios []ScenarioReport `json:"scenarios"`
}

type ReportDoc struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Passed      int             `json:"passed"`
	Failed      int             `json:"failed"`
	Skipped     int             `json:"skipped"`
	Features    []FeatureReport `json:"features"`
}

type collector struct {
	mu      sync.Mutex
	started map[string]time.Time
	doc     ReportDoc
}

// Report collects scenario outcomes from the After hook.
var Report = &collector{}

func (c *collector) record(s *godog.Scenario, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	status, msg := "passed", ""
	switch {
	case err == nil:
	case errors.Is(err, godog.ErrPending):
		status = "skipped"
	case errors.Is(err, godog.ErrUndefined):
		status, msg = "failed", "undefined step: no step definition matches (implement it in backend/steps)"
	default:
		status, msg = "failed", err.Error()
	}
	idx := -1
	for i, f := range c.doc.Features {
		if f.Path == s.Uri {
			idx = i
		}
	}
	if idx < 0 {
		c.doc.Features = append(c.doc.Features, FeatureReport{Name: featureName(s.Uri), Path: s.Uri})
		idx = len(c.doc.Features) - 1
	}
	c.doc.Features[idx].Scenarios = append(c.doc.Features[idx].Scenarios, ScenarioReport{Name: s.Name, Status: status, Error: msg})
	switch status {
	case "passed":
		c.doc.Passed++
	case "failed":
		c.doc.Failed++
	default:
		c.doc.Skipped++
	}
}

func featureName(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return path
	}
	for _, l := range strings.Split(string(b), "\n") {
		if n, ok := strings.CutPrefix(strings.TrimSpace(l), "Feature:"); ok {
			return strings.TrimSpace(n)
		}
	}
	return path
}

// Write saves the report as JSON to path.
func (c *collector) Write(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.doc.GeneratedAt = time.Now().UTC()
	b, err := json.MarshalIndent(c.doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
