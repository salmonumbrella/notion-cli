package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type fakePageSchemaClient struct {
	db      *notion.Database
	dbErr   error
	ds      *notion.DataSource
	dsErr   error
	dbCalls []string
	dsCalls []string
}

func (f *fakePageSchemaClient) GetDatabase(_ context.Context, id string) (*notion.Database, error) {
	f.dbCalls = append(f.dbCalls, id)
	return f.db, f.dbErr
}

func (f *fakePageSchemaClient) GetDataSource(_ context.Context, id string) (*notion.DataSource, error) {
	f.dsCalls = append(f.dsCalls, id)
	return f.ds, f.dsErr
}

func TestFindTitlePropertyNameFromDataSource(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		want       string
	}{
		{
			name: "finds title property",
			properties: map[string]interface{}{
				"Title": map[string]interface{}{"type": "title"},
				"DRI":   map[string]interface{}{"type": "people"},
			},
			want: "Title",
		},
		{
			name: "returns default when no title property",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{"type": "status"},
			},
			want: "title",
		},
		{
			name:       "returns default for nil properties",
			properties: nil,
			want:       "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTitlePropertyNameFromDataSource(tt.properties)
			if got != tt.want {
				t.Fatalf("findTitlePropertyNameFromDataSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveTitlePropertyNameForPageCreate_DataSourceParent(t *testing.T) {
	client := &fakePageSchemaClient{
		ds: &notion.DataSource{
			Properties: map[string]interface{}{
				"Task": map[string]interface{}{"type": "title"},
			},
		},
	}

	got, err := resolveTitlePropertyNameForPageCreate(context.Background(), client, "", "page", "ds_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Task" {
		t.Fatalf("title property = %q, want %q", got, "Task")
	}
	if len(client.dsCalls) != 1 || client.dsCalls[0] != "ds_123" {
		t.Fatalf("expected one GetDataSource(ds_123) call, got %+v", client.dsCalls)
	}
	if len(client.dbCalls) != 0 {
		t.Fatalf("expected no GetDatabase calls, got %+v", client.dbCalls)
	}
}

func TestResolveTitlePropertyNameForPageCreate_DatabaseParent(t *testing.T) {
	client := &fakePageSchemaClient{
		db: &notion.Database{
			Properties: map[string]map[string]interface{}{
				"Name": {"type": "title"},
			},
		},
	}

	got, err := resolveTitlePropertyNameForPageCreate(context.Background(), client, "db_123", "database", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Name" {
		t.Fatalf("title property = %q, want %q", got, "Name")
	}
	if len(client.dbCalls) != 1 || client.dbCalls[0] != "db_123" {
		t.Fatalf("expected one GetDatabase(db_123) call, got %+v", client.dbCalls)
	}
}

func TestResolveTitlePropertyNameForPageCreate_PageParentDefault(t *testing.T) {
	client := &fakePageSchemaClient{}

	got, err := resolveTitlePropertyNameForPageCreate(context.Background(), client, "page_123", "page", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "title" {
		t.Fatalf("title property = %q, want default %q", got, "title")
	}
	if len(client.dbCalls) != 0 || len(client.dsCalls) != 0 {
		t.Fatalf("expected no schema calls, db=%+v ds=%+v", client.dbCalls, client.dsCalls)
	}
}

func TestResolveTitlePropertyNameForPageCreate_DatabaseHintForDataSourceID(t *testing.T) {
	const id = "1d4eaecc-f764-8195-a931-000bd5878943"
	client := &fakePageSchemaClient{
		dbErr: errors.New("GET /databases/" + id + " (404): object_not_found"),
		ds: &notion.DataSource{
			ID: id,
			Title: []notion.RichText{
				{PlainText: "Issue Tracker"},
			},
		},
	}

	_, err := resolveTitlePropertyNameForPageCreate(context.Background(), client, id, "database", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ue *clierrors.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UserError hint, got %T: %v", err, err)
	}
	if !strings.Contains(ue.Suggestion, "ntn datasource get "+id) {
		t.Fatalf("expected datasource hint in suggestion, got: %s", ue.Suggestion)
	}
}
