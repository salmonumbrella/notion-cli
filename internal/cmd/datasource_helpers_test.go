package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type fakeResolveDataSourceClient struct {
	db      *notion.Database
	dbErr   error
	ds      *notion.DataSource
	dsErr   error
	dbCalls int
	dsCalls int
}

func (f *fakeResolveDataSourceClient) GetDatabase(_ context.Context, _ string) (*notion.Database, error) {
	f.dbCalls++
	return f.db, f.dbErr
}

func (f *fakeResolveDataSourceClient) GetDataSource(_ context.Context, _ string) (*notion.DataSource, error) {
	f.dsCalls++
	return f.ds, f.dsErr
}

func TestResolveDataSourceID_WithSingleDataSource(t *testing.T) {
	client := &fakeResolveDataSourceClient{
		db: &notion.Database{
			DataSources: []notion.DataSourceRef{{ID: "ds_1", Name: "Main"}},
		},
	}

	got, err := resolveDataSourceID(context.Background(), client, "db_1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ds_1" {
		t.Fatalf("resolveDataSourceID() = %q, want %q", got, "ds_1")
	}
	if client.dbCalls != 1 {
		t.Fatalf("expected 1 GetDatabase call, got %d", client.dbCalls)
	}
}

func TestResolveDataSourceID_HintsWhenGivenDataSourceIDAsDatabaseID(t *testing.T) {
	const id = "1d4eaecc-f764-8195-a931-000bd5878943"
	client := &fakeResolveDataSourceClient{
		dbErr: errors.New("GET /databases/" + id + " (404): object_not_found"),
		ds: &notion.DataSource{
			ID: id,
			Title: []notion.RichText{
				{PlainText: "Issue Tracker"},
			},
		},
	}

	_, err := resolveDataSourceID(context.Background(), client, id, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ue *clierrors.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UserError hint, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Suggestion, "ntn datasource get "+id) {
		t.Fatalf("expected datasource hint, got: %s", ue.Suggestion)
	}
	if client.dsCalls != 1 {
		t.Fatalf("expected 1 GetDataSource lookup for hinting, got %d", client.dsCalls)
	}
}
