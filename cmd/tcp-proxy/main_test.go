package main

import "testing"

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
