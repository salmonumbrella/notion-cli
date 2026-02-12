package cmd

import (
	"context"

	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func clientFromContext(ctx context.Context) (*notion.Client, error) {
	token, err := GetTokenFromContext(ctx)
	if err != nil {
		return nil, errors.AuthRequiredError(err)
	}
	return NewNotionClient(ctx, token), nil
}
