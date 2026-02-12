package cmd

import (
	"fmt"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

func wrapAPIError(err error, action, entityType, identifier string) error {
	if err == nil {
		return nil
	}
	mapped := clierrors.APINotFoundError(err, entityType, identifier)
	if mapped != err {
		return mapped
	}
	if action == "" {
		return err
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}
