package cmd

import (
	"fmt"

	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

// wrapAPIError maps Notion API errors to user-friendly types. If the error
// indicates a 404, it returns a NotFoundError with the entity type and ID.
// Otherwise it wraps the error with "failed to <action>: ..." context.
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
