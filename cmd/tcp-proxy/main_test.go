package main

import (
	"os"
	"testing"
)

func TestParseObservedPort(t *testing.T) {
	port, err := parseObservedPort(":8090")
	if err != nil {
		t.Fatalf("parseObservedPort failed: %v", err)
	}
	if port != 8090 {
		t.Fatalf("unexpected port: %d", port)
	}

	port, err = parseObservedPort("127.0.0.1:9000")
	if err != nil {
		t.Fatalf("parseObservedPort host failed: %v", err)
	}
	if port != 9000 {
		t.Fatalf("unexpected host port: %d", port)
	}
}

func TestParseObservedPortRejectsInvalidPort(t *testing.T) {
	if _, err := parseObservedPort("abc"); err == nil {
		t.Fatal("expected parseObservedPort to reject non-numeric port")
	}
	if _, err := parseObservedPort(":70000"); err == nil {
		t.Fatal("expected parseObservedPort to reject out-of-range port")
	}
}

func TestIsBackgroundChild(t *testing.T) {
	original, existed := os.LookupEnv(backgroundChildEnv)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(backgroundChildEnv, original)
			return
		}
		_ = os.Unsetenv(backgroundChildEnv)
	})

	_ = os.Unsetenv(backgroundChildEnv)
	if isBackgroundChild() {
		t.Fatal("unset environment must not be treated as a background child")
	}
	if err := os.Setenv(backgroundChildEnv, "1"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	if !isBackgroundChild() {
		t.Fatal("background child marker was not recognized")
	}
}
