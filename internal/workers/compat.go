package workers

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Compat stores pinned versions/commits used by workers integration.
type Compat struct {
	WorkersCLIVersion string `json:"workers_cli_version"`
	TemplateRepo      string `json:"template_repo"`
	TemplateCommit    string `json:"template_commit"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

//go:embed compat.json
var compatJSON []byte

var (
	compatOnce sync.Once
	compatCfg  Compat
	compatErr  error
)

// Current returns the pinned workers compatibility configuration.
func Current() (Compat, error) {
	compatOnce.Do(func() {
		compatErr = json.Unmarshal(compatJSON, &compatCfg)
		if compatErr != nil {
			compatErr = fmt.Errorf("parse workers compat config: %w", compatErr)
			return
		}
		compatCfg.WorkersCLIVersion = strings.TrimSpace(compatCfg.WorkersCLIVersion)
		compatCfg.TemplateRepo = strings.TrimSpace(compatCfg.TemplateRepo)
		compatCfg.TemplateCommit = strings.TrimSpace(compatCfg.TemplateCommit)
		if compatCfg.WorkersCLIVersion == "" {
			compatErr = fmt.Errorf("workers compat config missing workers_cli_version")
			return
		}
		if compatCfg.TemplateRepo == "" {
			compatErr = fmt.Errorf("workers compat config missing template_repo")
			return
		}
		if compatCfg.TemplateCommit == "" {
			compatErr = fmt.Errorf("workers compat config missing template_commit")
			return
		}
	})
	if compatErr != nil {
		return Compat{}, compatErr
	}
	return compatCfg, nil
}
