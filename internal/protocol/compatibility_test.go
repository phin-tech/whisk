package protocol

import (
	"errors"
	"strings"
	"testing"
)

func TestCompatibilityResponseDaemonProtocolVersionFallsBackToAPIVersion(t *testing.T) {
	response := CompatibilityResponse{APIVersion: ProtocolVersion - 1}
	if got := response.DaemonProtocolVersion(); got != ProtocolVersion-1 {
		t.Fatalf("daemon protocol version = %d, want %d", got, ProtocolVersion-1)
	}
	response.ProtocolVersion = ProtocolVersion
	if got := response.DaemonProtocolVersion(); got != ProtocolVersion {
		t.Fatalf("daemon protocol version = %d, want %d", got, ProtocolVersion)
	}
}

func TestCheckCompatibilityDecisions(t *testing.T) {
	clientVersion := ProtocolVersion
	previousVersion := ProtocolVersion - 1
	nextVersion := ProtocolVersion + 1

	tests := []struct {
		name       string
		policy     CompatibilityPolicy
		response   CompatibilityResponse
		compatible bool
	}{
		{
			name:       "current protocol",
			response:   CompatibilityResponse{APIVersion: clientVersion, ProtocolVersion: clientVersion},
			compatible: true,
		},
		{
			name:       "legacy api version fallback",
			response:   CompatibilityResponse{APIVersion: clientVersion},
			compatible: true,
		},
		{
			name: "client supports previous daemon protocol",
			policy: CompatibilityPolicy{
				ClientProtocolVersion:           clientVersion,
				SupportedDaemonProtocolVersions: []int{previousVersion},
			},
			response:   CompatibilityResponse{APIVersion: previousVersion, ProtocolVersion: previousVersion},
			compatible: true,
		},
		{
			name:       "previous daemon protocol unsupported",
			response:   CompatibilityResponse{APIVersion: previousVersion, ProtocolVersion: previousVersion},
			compatible: false,
		},
		{
			name: "newer daemon supports this client protocol",
			response: CompatibilityResponse{
				APIVersion:                        nextVersion,
				ProtocolVersion:                   nextVersion,
				SupportedPreviousProtocolVersions: []int{clientVersion},
			},
			compatible: true,
		},
		{
			name:       "newer daemon protocol unsupported",
			response:   CompatibilityResponse{APIVersion: nextVersion, ProtocolVersion: nextVersion},
			compatible: false,
		},
		{
			name:       "missing daemon protocol unsupported",
			response:   CompatibilityResponse{},
			compatible: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := tt.policy.Check(tt.response)
			if tt.policy.ClientProtocolVersion == 0 && len(tt.policy.SupportedDaemonProtocolVersions) == 0 {
				decision = CheckCompatibility(tt.response)
			}
			if decision.Compatible != tt.compatible {
				t.Fatalf("compatible = %v, want %v; decision = %#v", decision.Compatible, tt.compatible, decision)
			}
			if got, want := decision.ClientProtocolVersion, clientVersion; got != want {
				t.Fatalf("client protocol version = %d, want %d", got, want)
			}
			if !containsProtocolVersion(decision.SupportedDaemonProtocolVersions, clientVersion) {
				t.Fatalf("supported daemon versions missing current client protocol: %#v", decision)
			}
		})
	}
}

func TestEnsureCompatibleReturnsTypedActionableError(t *testing.T) {
	err := EnsureCompatible(CompatibilityResponse{
		APIVersion:      ProtocolVersion + 1,
		ProtocolVersion: ProtocolVersion + 1,
	})
	if err == nil {
		t.Fatalf("expected compatibility error")
	}
	var compatibilityErr *CompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("expected CompatibilityError, got %T", err)
	}
	for _, want := range []string{
		"daemon protocol",
		"client protocol",
		"supported daemon protocols",
		"Upgrade Whisk",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err.Error(), want)
		}
	}
}

func TestSupportedDaemonProtocolVersionsReturnsCopy(t *testing.T) {
	versions := SupportedDaemonProtocolVersions()
	if len(versions) == 0 || versions[0] != ProtocolVersion {
		t.Fatalf("supported daemon protocol versions = %#v", versions)
	}
	versions[0] = ProtocolVersion + 100
	if got := SupportedDaemonProtocolVersions()[0]; got != ProtocolVersion {
		t.Fatalf("supported daemon protocol versions leaked mutable backing array, got %d", got)
	}
}
