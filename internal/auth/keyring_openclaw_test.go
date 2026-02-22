package auth

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyringFileDir_UsesNotionCredentialsDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "credentials")
	t.Setenv(CredentialsDirEnvVarName, base)
	t.Setenv(SharedCredentialsDirEnvVarName, filepath.Join(t.TempDir(), "shared"))

	got := keyringFileDir()
	want := filepath.Join(base, ServiceName, "keyring")
	if got != want {
		t.Fatalf("keyringFileDir() = %q, want %q", got, want)
	}
}

func TestKeyringFileDir_UsesSharedCredentialsDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "shared-credentials")
	t.Setenv(CredentialsDirEnvVarName, "")
	t.Setenv(SharedCredentialsDirEnvVarName, base)

	got := keyringFileDir()
	want := filepath.Join(base, ServiceName, "keyring")
	if got != want {
		t.Fatalf("keyringFileDir() = %q, want %q", got, want)
	}
}

func TestKeyringFileDir_DefaultSuffix(t *testing.T) {
	t.Setenv(CredentialsDirEnvVarName, "")
	t.Setenv(SharedCredentialsDirEnvVarName, "")

	got := keyringFileDir()
	wantSuffix := filepath.Join(ServiceName, "keyring")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("keyringFileDir() = %q, want suffix %q", got, wantSuffix)
	}
}
