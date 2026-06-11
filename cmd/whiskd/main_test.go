package main

import "testing"

func TestValidateListenAddrAllowsLoopbackOnly(t *testing.T) {
	allowed := []string{
		"127.0.0.1:8787",
		"localhost:8787",
		"[::1]:8787",
	}
	for _, addr := range allowed {
		if err := validateListenAddr(addr); err != nil {
			t.Fatalf("expected %s to be allowed: %v", addr, err)
		}
	}

	blocked := []string{
		"0.0.0.0:8787",
		"192.168.1.10:8787",
		"example.com:8787",
	}
	for _, addr := range blocked {
		if err := validateListenAddr(addr); err == nil {
			t.Fatalf("expected %s to be blocked", addr)
		}
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("WHISK_TEST_ENV", "set")
	if got := envOrDefault("WHISK_TEST_ENV", "fallback"); got != "set" {
		t.Fatalf("env value = %q", got)
	}
	if got := envOrDefault("WHISK_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("fallback value = %q", got)
	}
}
