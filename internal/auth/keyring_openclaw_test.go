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

func TestKeyringFilePassword_Priority(t *testing.T) {
	t.Setenv(KeyringPasswordEnvVarName, "primary")
	t.Setenv(SharedKeyringPasswordEnvVarName, "shared")
	t.Setenv(OpenClawKeyringPasswordEnvVarName, "openclaw")

	if got := keyringFilePassword(); got != "primary" {
		t.Fatalf("keyringFilePassword() = %q, want %q", got, "primary")
	}
}

func TestKeyringFilePassword_SharedFallback(t *testing.T) {
	t.Setenv(KeyringPasswordEnvVarName, "")
	t.Setenv(SharedKeyringPasswordEnvVarName, "shared")
	t.Setenv(OpenClawKeyringPasswordEnvVarName, "")

	if got := keyringFilePassword(); got != "shared" {
		t.Fatalf("keyringFilePassword() = %q, want %q", got, "shared")
	}
}

func TestShouldForceFileBackend(t *testing.T) {
	if !shouldForceFileBackend("linux", "") {
		t.Fatalf("expected linux+empty dbus to force file backend")
	}
	if shouldForceFileBackend("linux", "unix:path=/tmp/dbus") {
		t.Fatalf("expected linux+dbus set to avoid forcing file backend")
	}
	if shouldForceFileBackend("darwin", "") {
		t.Fatalf("expected non-linux to avoid forcing file backend")
	}
}
