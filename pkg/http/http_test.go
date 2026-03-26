package http

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseRequestLine(t *testing.T) {
	tests := []struct {
		line      string
		method    string
		uri       string
		proto     string
		path      string
		query     string
	}{
		{"GET / HTTP/1.1", "GET", "/", "HTTP/1.1", "/", ""},
		{"GET /index.html HTTP/1.1", "GET", "/index.html", "HTTP/1.1", "/index.html", ""},
		{"GET /api/users?id=123 HTTP/1.1", "GET", "/api/users?id=123", "HTTP/1.1", "/api/users", "id=123"},
		{"POST /submit HTTP/1.1", "POST", "/submit", "HTTP/1.1", "/submit", ""},
	}

	for _, tt := range tests {
		req := &Request{}
		err := req.ParseRequestLine(tt.line)
		if err != nil {
			t.Errorf("ParseRequestLine(%q) error: %v", tt.line, err)
			continue
		}

		if req.Method != tt.method {
			t.Errorf("Method: expected %q, got %q", tt.method, req.Method)
		}
		if req.URI != tt.uri {
			t.Errorf("URI: expected %q, got %q", tt.uri, req.URI)
		}
		if req.Proto != tt.proto {
			t.Errorf("Proto: expected %q, got %q", tt.proto, req.Proto)
		}
		if req.Path != tt.path {
			t.Errorf("Path: expected %q, got %q", tt.path, req.Path)
		}
		if req.Query != tt.query {
			t.Errorf("Query: expected %q, got %q", tt.query, req.Query)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	lines := []string{
		"Host: localhost",
		"Content-Type: application/json",
		"Accept: */*",
		"",
		"body content",
	}

	req := &Request{}
	err := req.ParseHeaders(lines)

	if err != nil {
		t.Errorf("ParseHeaders error: %v", err)
	}

	if req.Headers["Host"] != "localhost" {
		t.Errorf("Host: expected 'localhost', got %q", req.Headers["Host"])
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type: expected 'application/json', got %q", req.Headers["Content-Type"])
	}
	if req.Host != "localhost" {
		t.Errorf("req.Host: expected 'localhost', got %q", req.Host)
	}
}

func TestStatusText(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{301, "Moved Permanently"},
	}

	for _, tt := range tests {
		result := StatusText(tt.code)
		if result != tt.expected {
			t.Errorf("StatusText(%d): expected %q, got %q", tt.code, tt.expected, result)
		}
	}
}