package config

import (
	"testing"
)

func TestScanner(t *testing.T) {
	input := []byte(`
events {
    worker_connections 1024;
}
http {
    server {
        listen 80;
        server_name localhost;
    }
}
`)

	scanner := NewScanner(input)
	
	tests := []struct {
		expectedType TokenType
		expectedValue string
	}{
		{TokenIdent, "events"},
		{TokenLeftBrace, "{"},
		{TokenIdent, "worker_connections"},
		{TokenIdent, "1024"},
		{TokenSemicolon, ";"},
		{TokenRightBrace, "}"},
		{TokenIdent, "http"},
		{TokenLeftBrace, "{"},
		{TokenIdent, "server"},
		{TokenLeftBrace, "{"},
		{TokenIdent, "listen"},
		{TokenIdent, "80"},
		{TokenSemicolon, ";"},
		{TokenRightBrace, "}"},
		{TokenRightBrace, "}"},
	}

	for i, tt := range tests {
		tok := scanner.Next()
		if tok.Type != tt.expectedType {
			t.Errorf("Token %d: expected type %v, got %v", i, tt.expectedType, tok.Type)
		}
		if tok.Value != tt.expectedValue {
			t.Errorf("Token %d: expected value %q, got %q", i, tt.expectedValue, tok.Value)
		}
	}
}

func TestScannerString(t *testing.T) {
	input := []byte(`"hello world"`)

	scanner := NewScanner(input)
	tok := scanner.Next()

	if tok.Type != TokenString {
		t.Errorf("expected TokenString, got %v", tok.Type)
	}
	if tok.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", tok.Value)
	}
}

func TestScannerComment(t *testing.T) {
	input := []byte(`# this is a comment
events {
`)

	scanner := NewScanner(input)
	tok := scanner.Next()

	if tok.Type != TokenIdent {
		t.Errorf("expected TokenIdent, got %v", tok.Type)
	}
	if tok.Value != "events" {
		t.Errorf("expected 'events', got %q", tok.Value)
	}
}