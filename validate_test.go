package main

import "testing"

func TestSanitizeAppName(t *testing.T) {
	ok, err := SanitizeAppName("MyApp")
	if err != nil || ok != "MyApp" {
		t.Fatalf("expected MyApp, got %q err=%v", ok, err)
	}
	if _, err := SanitizeAppName("../evil"); err == nil {
		t.Fatal("expected error for path traversal")
	}
	if _, err := SanitizeAppName("CON"); err == nil {
		t.Fatal("expected error for reserved name")
	}
	if _, err := SanitizeAppName("bad:name"); err == nil {
		t.Fatal("expected error for invalid char")
	}
	if _, err := SanitizeAppName(""); err == nil {
		t.Fatal("expected error for empty")
	}
}

func TestValidateProxyRule(t *testing.T) {
	err := ValidateProxyRule(ProxyRule{Path: "/api/", Target: "http://localhost:8080/", Enabled: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateProxyRule(ProxyRule{Path: "/api/", Target: "ftp://x", Enabled: true}, 0)
	if err == nil {
		t.Fatal("expected scheme error")
	}
	err = ValidateProxyRule(ProxyRule{Path: "/api/", Target: "http://localhost:8080/", Rewrite: "v2", Enabled: true}, 0)
	if err == nil {
		t.Fatal("expected rewrite must start with /")
	}
}