package gcloud

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMessage(t *testing.T) {
	raw, err := buildMessage("someone@example.com", "Hej på dig", "Week summary")
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	msg := string(decoded)

	if !strings.Contains(msg, "To: <someone@example.com>\r\n") {
		t.Errorf("missing To header:\n%s", msg)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?q?") {
		t.Errorf("subject not encoded:\n%s", msg)
	}
	wantBody := base64.StdEncoding.EncodeToString([]byte("Week summary"))
	if !strings.HasSuffix(msg, "\r\n\r\n"+wantBody) {
		t.Errorf("unexpected body:\n%s", msg)
	}
}

func TestBuildMessageRejectsBadInput(t *testing.T) {
	if _, err := buildMessage("not an address", "hi", "body"); err == nil {
		t.Error("expected error for invalid address")
	}
	if _, err := buildMessage("a@example.com", "hi\r\nBcc: x@example.com", "body"); err == nil {
		t.Error("expected error for subject with newline")
	}
}
