package notion

import (
	"strings"
	"testing"
)

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test_webhook_signing_key_for_unit_tests"
	body := []byte(`{"type":"page.content_updated","data":{"page_id":"abc123"}}`)

	signature := ComputeWebhookSignature(secret, body)

	if !VerifyWebhookSignature(secret, body, signature) {
		t.Error("expected valid signature to verify")
	}
}

func TestVerifyWebhookSignature_Invalid(t *testing.T) {
	secret := "secret_test"
	body := []byte(`{"type":"page.content_updated"}`)
	wrongSig := "sha256=invalidhash"

	if VerifyWebhookSignature(secret, body, wrongSig) {
		t.Error("expected invalid signature to fail verification")
	}
}

func TestParseWebhookEvent_PageContentUpdated(t *testing.T) {
	payload := []byte(`{
		"type": "page.content_updated",
		"timestamp": "2024-01-15T10:30:00.000Z",
		"data": {
			"page_id": "page123",
			"workspace_id": "ws456"
		}
	}`)

	event, err := ParseWebhookEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "page.content_updated" {
		t.Errorf("expected type 'page.content_updated', got %q", event.Type)
	}
	if event.Data.PageID != "page123" {
		t.Errorf("expected page_id 'page123', got %q", event.Data.PageID)
	}
}

func TestComputeWebhookSignature(t *testing.T) {
	secret := "test_secret"
	body := []byte("test body")

	sig := ComputeWebhookSignature(secret, body)

	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("expected signature to start with 'sha256=', got %q", sig)
	}
}

func TestIsVerificationRequest_True(t *testing.T) {
	payload := []byte(`{"verification_token": "secret_abc123"}`)

	if !IsVerificationRequest(payload) {
		t.Error("expected verification request to be detected")
	}
}

func TestIsVerificationRequest_False(t *testing.T) {
	payload := []byte(`{"type": "page.content_updated"}`)

	if IsVerificationRequest(payload) {
		t.Error("expected non-verification request")
	}
}

func TestParseWebhookVerification(t *testing.T) {
	payload := []byte(`{"verification_token": "secret_test123"}`)

	req, err := ParseWebhookVerification(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.VerificationToken != "secret_test123" {
		t.Errorf("expected token 'secret_test123', got %q", req.VerificationToken)
	}
}

func TestVerifyWebhookSignature_EmptySecret(t *testing.T) {
	secret := ""
	body := []byte(`{"type":"page.content_updated"}`)

	signature := ComputeWebhookSignature(secret, body)

	if !VerifyWebhookSignature(secret, body, signature) {
		t.Error("expected signature to verify even with empty secret")
	}
}

func TestVerifyWebhookSignature_EmptyBody(t *testing.T) {
	secret := "secret_test"

	// Test with nil body
	nilSignature := ComputeWebhookSignature(secret, nil)
	if !VerifyWebhookSignature(secret, nil, nilSignature) {
		t.Error("expected signature to verify with nil body")
	}

	// Test with zero-length body
	emptySignature := ComputeWebhookSignature(secret, []byte{})
	if !VerifyWebhookSignature(secret, []byte{}, emptySignature) {
		t.Error("expected signature to verify with empty body")
	}
}

func TestParseWebhookEvent_InvalidJSON(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "completely invalid JSON",
			payload: []byte(`{invalid json`),
		},
		{
			name:    "truncated JSON",
			payload: []byte(`{"type":"page.content_updated","data":`),
		},
		{
			name:    "non-JSON text",
			payload: []byte(`this is not json at all`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseWebhookEvent(tc.payload)
			if err == nil {
				t.Errorf("expected error parsing invalid JSON, got none")
			}
		})
	}
}

func TestParseWebhookEvent_EmptyBody(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "nil body",
			payload: nil,
		},
		{
			name:    "zero-length body",
			payload: []byte{},
		},
		{
			name:    "empty JSON object",
			payload: []byte(`{}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseWebhookEvent(tc.payload)

			// Empty JSON object is valid JSON, so it shouldn't error
			if tc.name == "empty JSON object" {
				if err != nil {
					t.Errorf("unexpected error for empty JSON object: %v", err)
				}
				if event == nil {
					t.Error("expected non-nil event for empty JSON object")
				}
			} else {
				// nil and zero-length should error
				if err == nil {
					t.Error("expected error parsing empty body, got none")
				}
			}
		})
	}
}

func TestComputeWebhookSignature_Consistency(t *testing.T) {
	secret := "secret_consistency_test"
	body := []byte(`{"type":"page.content_updated","data":{"page_id":"abc123"}}`)

	// Compute signature multiple times
	sig1 := ComputeWebhookSignature(secret, body)
	sig2 := ComputeWebhookSignature(secret, body)
	sig3 := ComputeWebhookSignature(secret, body)

	// All signatures should be identical
	if sig1 != sig2 {
		t.Errorf("signatures are inconsistent: sig1=%q, sig2=%q", sig1, sig2)
	}
	if sig1 != sig3 {
		t.Errorf("signatures are inconsistent: sig1=%q, sig3=%q", sig1, sig3)
	}
	if sig2 != sig3 {
		t.Errorf("signatures are inconsistent: sig2=%q, sig3=%q", sig2, sig3)
	}

	// Verify all signatures work for verification
	if !VerifyWebhookSignature(secret, body, sig1) {
		t.Error("sig1 failed verification")
	}
	if !VerifyWebhookSignature(secret, body, sig2) {
		t.Error("sig2 failed verification")
	}
	if !VerifyWebhookSignature(secret, body, sig3) {
		t.Error("sig3 failed verification")
	}
}
