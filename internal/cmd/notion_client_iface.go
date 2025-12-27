package cmd

import (
	"context"
	"net/http"

	"github.com/salmonumbrella/notion-cli/internal/notion"
)

// databaseGetter describes the minimal database lookup needed by helpers.
type databaseGetter interface {
	GetDatabase(ctx context.Context, id string) (*notion.Database, error)
}

// pageSchemaGetter describes the schema lookups needed when creating pages
// with shorthand flags against database/data source parents.
type pageSchemaGetter interface {
	databaseGetter
	dataSourceGetter
}

// blockChildrenReader describes the block children retrieval used by export/duplicate helpers.
type blockChildrenReader interface {
	GetBlockChildren(ctx context.Context, blockID string, opts *notion.BlockChildrenOptions) (*notion.BlockList, error)
}

// blockChildrenWriter describes the block children append operation used by duplicate helpers.
type blockChildrenWriter interface {
	AppendBlockChildren(ctx context.Context, blockID string, req *notion.AppendBlockChildrenRequest) (*notion.BlockList, error)
}

// rawRequester describes the raw API request used by api request helpers.
type rawRequester interface {
	DoRawRequest(ctx context.Context, method, path string, body []byte, headers http.Header) (*notion.RawResponse, error)
}

var (
	_ blockChildrenReader = (*notion.Client)(nil)
	_ blockChildrenWriter = (*notion.Client)(nil)
	_ rawRequester        = (*notion.Client)(nil)
	_ pageSchemaGetter    = (*notion.Client)(nil)
)
