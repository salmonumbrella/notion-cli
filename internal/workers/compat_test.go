package workers

import "testing"

func TestCurrent(t *testing.T) {
	cfg, err := Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if cfg.WorkersCLIVersion == "" {
		t.Fatal("workers_cli_version should not be empty")
	}
	if cfg.TemplateRepo == "" {
		t.Fatal("template_repo should not be empty")
	}
	if cfg.TemplateCommit == "" {
		t.Fatal("template_commit should not be empty")
	}
}
