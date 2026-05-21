package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScriptRunRequestAllowedRequiresRockOSHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)

	if scriptRunRequestAllowed(request) {
		t.Fatal("script run request without Rock-OS header was allowed")
	}
}

func TestScriptRunRequestAllowedRejectsCrossOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/scripts/run", nil)
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "https://example.com")

	if scriptRunRequestAllowed(request) {
		t.Fatal("cross-origin script run request was allowed")
	}
}

func TestScriptRunRequestAllowedAcceptsSameOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://192.168.1.2:8000/api/scripts/run", nil)
	request.Header.Set("X-Rock-OS-Requested", "true")
	request.Header.Set("Origin", "http://192.168.1.2:8000")
	request.Header.Set("Referer", "http://192.168.1.2:8000/scripts.html")

	if !scriptRunRequestAllowed(request) {
		t.Fatal("same-origin script run request was rejected")
	}
}
