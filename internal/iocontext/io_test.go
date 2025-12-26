// internal/iocontext/io_test.go
package iocontext

import (
	"bytes"
	"context"
	"testing"
)

func TestWithIO_DefaultsToNil(t *testing.T) {
	ctx := context.Background()
	out := Stdout(ctx)
	if out != nil {
		t.Errorf("expected nil stdout for empty context, got %v", out)
	}
}

func TestWithIO_InjectsWriters(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := WithIO(context.Background(), &stdout, &stderr)

	out := Stdout(ctx)
	if out != &stdout {
		t.Errorf("expected injected stdout")
	}

	errOut := Stderr(ctx)
	if errOut != &stderr {
		t.Errorf("expected injected stderr")
	}
}
