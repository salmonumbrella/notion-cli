package iocontext

import (
	"bytes"
	"context"
	"testing"
)

func TestStdout_ReturnsNilForEmptyContext(t *testing.T) {
	ctx := context.Background()
	out := Stdout(ctx)
	if out != nil {
		t.Errorf("expected nil stdout for empty context, got %v", out)
	}
}

func TestStderr_ReturnsNilForEmptyContext(t *testing.T) {
	ctx := context.Background()
	errOut := Stderr(ctx)
	if errOut != nil {
		t.Errorf("expected nil stderr for empty context, got %v", errOut)
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

func TestWithIO_NilWritersStoredAsNil(t *testing.T) {
	ctx := WithIO(context.Background(), nil, nil)

	out := Stdout(ctx)
	if out != nil {
		t.Errorf("expected nil stdout when nil injected, got %v", out)
	}

	errOut := Stderr(ctx)
	if errOut != nil {
		t.Errorf("expected nil stderr when nil injected, got %v", errOut)
	}
}

func TestStdoutOrDefault_FallsBackWhenNoWriter(t *testing.T) {
	var defaultWriter bytes.Buffer
	ctx := context.Background()

	result := StdoutOrDefault(ctx, &defaultWriter)
	if result != &defaultWriter {
		t.Errorf("expected default writer when context has no stdout")
	}
}

func TestStdoutOrDefault_ReturnsContextWriter(t *testing.T) {
	var contextWriter, defaultWriter bytes.Buffer
	ctx := WithIO(context.Background(), &contextWriter, nil)

	result := StdoutOrDefault(ctx, &defaultWriter)
	if result != &contextWriter {
		t.Errorf("expected context writer, got default")
	}
}

func TestStderrOrDefault_FallsBackWhenNoWriter(t *testing.T) {
	var defaultWriter bytes.Buffer
	ctx := context.Background()

	result := StderrOrDefault(ctx, &defaultWriter)
	if result != &defaultWriter {
		t.Errorf("expected default writer when context has no stderr")
	}
}

func TestStderrOrDefault_ReturnsContextWriter(t *testing.T) {
	var contextWriter, defaultWriter bytes.Buffer
	ctx := WithIO(context.Background(), nil, &contextWriter)

	result := StderrOrDefault(ctx, &defaultWriter)
	if result != &contextWriter {
		t.Errorf("expected context writer, got default")
	}
}
