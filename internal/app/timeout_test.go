package app

import "testing"

func TestActionTimeoutMsDefault(t *testing.T) {
	ms, err := actionTimeoutMs(GlobalFlags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms != 20000 {
		t.Fatalf("expected 20000ms, got %d", ms)
	}
}

func TestActionTimeoutMsParse(t *testing.T) {
	ms, err := actionTimeoutMs(GlobalFlags{Timeout: "5s"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms != 5000 {
		t.Fatalf("expected 5000ms, got %d", ms)
	}
}

func TestActionTimeoutMsInvalid(t *testing.T) {
	_, err := actionTimeoutMs(GlobalFlags{Timeout: "bad"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
