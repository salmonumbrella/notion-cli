package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

type fakeDataSourceGetter struct {
	ds     *notion.DataSource
	err    error
	called int
}

func (f *fakeDataSourceGetter) GetDataSource(context.Context, string) (*notion.DataSource, error) {
	f.called++
	return f.ds, f.err
}

func TestMaybeDataSourceHintForDatabaseNotFound(t *testing.T) {
	const id = "7700d334-e15f-4da6-994b-56f7d847a931"

	getter := &fakeDataSourceGetter{
		ds: &notion.DataSource{
			ID: id,
			Title: []notion.RichText{
				{PlainText: "Finance Invoices"},
			},
		},
	}

	dbErr := errors.New("GET /databases/" + id + " (404): object_not_found")
	err := maybeDataSourceHintForDatabaseNotFound(context.Background(), getter, dbErr, id)
	if err == nil {
		t.Fatal("expected hint error, got nil")
	}

	var ue *clierrors.UserError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UserError, got %T", err)
	}
	if getter.called != 1 {
		t.Fatalf("expected data source lookup to be called once, got %d", getter.called)
	}

	msg := ue.Suggestion
	if !strings.Contains(msg, "data source") {
		t.Fatalf("expected data source hint in suggestion, got: %s", msg)
	}
	if !strings.Contains(msg, "ntn datasource get "+id) {
		t.Fatalf("expected datasource get command in suggestion, got: %s", msg)
	}
	if !strings.Contains(msg, "Finance Invoices") {
		t.Fatalf("expected data source title in suggestion, got: %s", msg)
	}
}

func TestMaybeDataSourceHintForDatabaseNotFound_NonUUIDSkipsLookup(t *testing.T) {
	getter := &fakeDataSourceGetter{ds: &notion.DataSource{}}
	dbErr := errors.New("GET /databases/Finance Invoices (404): object_not_found")

	err := maybeDataSourceHintForDatabaseNotFound(context.Background(), getter, dbErr, "Finance Invoices")
	if err != nil {
		t.Fatalf("expected nil hint for non-UUID identifier, got: %v", err)
	}
	if getter.called != 0 {
		t.Fatalf("expected no data source lookup, got %d call(s)", getter.called)
	}
}

func TestMaybeDataSourceHintForDatabaseNotFound_NonNotFoundError(t *testing.T) {
	const id = "7700d334-e15f-4da6-994b-56f7d847a931"

	getter := &fakeDataSourceGetter{ds: &notion.DataSource{}}
	dbErr := errors.New("request timeout")

	err := maybeDataSourceHintForDatabaseNotFound(context.Background(), getter, dbErr, id)
	if err != nil {
		t.Fatalf("expected nil hint for non-404 error, got: %v", err)
	}
	if getter.called != 0 {
		t.Fatalf("expected no data source lookup, got %d call(s)", getter.called)
	}
}

func TestMaybeDataSourceHintForDatabaseNotFound_DataSourceLookupFails(t *testing.T) {
	const id = "7700d334-e15f-4da6-994b-56f7d847a931"

	getter := &fakeDataSourceGetter{err: errors.New("object_not_found")}
	dbErr := errors.New("GET /databases/" + id + " (404): object_not_found")

	err := maybeDataSourceHintForDatabaseNotFound(context.Background(), getter, dbErr, id)
	if err != nil {
		t.Fatalf("expected nil hint when datasource lookup fails, got: %v", err)
	}
	if getter.called != 1 {
		t.Fatalf("expected one data source lookup, got %d call(s)", getter.called)
	}
}
