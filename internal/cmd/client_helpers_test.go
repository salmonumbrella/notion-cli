package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/auth"
	clierrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

func TestClientFromContext(t *testing.T) {
	t.Run("no token returns AuthError", func(t *testing.T) {
		// Use a mock keyring with no token and ensure env var is unset.
		mock := auth.NewMockKeyringProvider()
		auth.SetProviderFunc(func() (auth.KeyringProvider, error) { return mock, nil })
		t.Cleanup(func() { auth.SetProviderFunc(nil) })
		t.Setenv("NOTION_TOKEN", "")

		ctx := context.Background()
		client, err := clientFromContext(ctx)

		if client != nil {
			t.Fatal("clientFromContext() returned non-nil client, want nil")
		}
		if err == nil {
			t.Fatal("clientFromContext() returned nil error, want AuthError")
		}
		var ae *clierrors.AuthError
		if !errors.As(err, &ae) {
			t.Fatalf("clientFromContext() error type = %T, want *errors.AuthError", err)
		}
	})

	t.Run("valid token returns client", func(t *testing.T) {
		mock := auth.NewMockKeyringProvider()
		mock.SetToken("ntn_test_token_12345")
		auth.SetProviderFunc(func() (auth.KeyringProvider, error) { return mock, nil })
		t.Cleanup(func() { auth.SetProviderFunc(nil) })

		ctx := context.Background()
		client, err := clientFromContext(ctx)
		if err != nil {
			t.Fatalf("clientFromContext() error = %v, want nil", err)
		}
		if client == nil {
			t.Fatal("clientFromContext() returned nil client, want non-nil")
		}
	})
}
