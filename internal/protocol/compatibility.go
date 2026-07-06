package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

func (r CompatibilityResponse) DaemonProtocolVersion() int {
	if r.ProtocolVersion != 0 {
		return r.ProtocolVersion
	}
	return r.APIVersion
}

func SupportedDaemonProtocolVersions() []int {
	return supportedProtocolVersions(ProtocolVersion, SupportedPreviousProtocolVersions())
}

type CompatibilityPolicy struct {
	ClientProtocolVersion           int
	SupportedDaemonProtocolVersions []int
}

type CompatibilityDecision struct {
	Compatible                              bool
	ClientProtocolVersion                   int
	DaemonProtocolVersion                   int
	SupportedDaemonProtocolVersions         []int
	DaemonSupportedPreviousProtocolVersions []int
}

func CheckCompatibility(response CompatibilityResponse) CompatibilityDecision {
	return DefaultCompatibilityPolicy().Check(response)
}

func EnsureCompatible(response CompatibilityResponse) error {
	return CheckCompatibility(response).Err()
}

func DefaultCompatibilityPolicy() CompatibilityPolicy {
	return CompatibilityPolicy{
		ClientProtocolVersion:           ProtocolVersion,
		SupportedDaemonProtocolVersions: SupportedDaemonProtocolVersions(),
	}
}

func (p CompatibilityPolicy) Check(response CompatibilityResponse) CompatibilityDecision {
	clientProtocolVersion := p.ClientProtocolVersion
	if clientProtocolVersion == 0 {
		clientProtocolVersion = ProtocolVersion
	}
	supportedDaemonVersions := supportedProtocolVersions(clientProtocolVersion, p.SupportedDaemonProtocolVersions)
	daemonProtocolVersion := response.DaemonProtocolVersion()
	daemonSupportedPrevious := append([]int(nil), response.SupportedPreviousProtocolVersions...)
	compatible := daemonProtocolVersion > 0 &&
		(containsProtocolVersion(supportedDaemonVersions, daemonProtocolVersion) ||
			containsProtocolVersion(daemonSupportedPrevious, clientProtocolVersion))
	return CompatibilityDecision{
		Compatible:                              compatible,
		ClientProtocolVersion:                   clientProtocolVersion,
		DaemonProtocolVersion:                   daemonProtocolVersion,
		SupportedDaemonProtocolVersions:         supportedDaemonVersions,
		DaemonSupportedPreviousProtocolVersions: daemonSupportedPrevious,
	}
}

func (d CompatibilityDecision) Err() error {
	if d.Compatible {
		return nil
	}
	return &CompatibilityError{Decision: d}
}

type CompatibilityError struct {
	Decision CompatibilityDecision
}

func (e *CompatibilityError) Error() string {
	return fmt.Sprintf(
		"daemon protocol %s is not supported by this app (client protocol %d; supported daemon protocols: %s; daemon supports previous client protocols: %s). Upgrade Whisk, run `whisk daemon restart`, or use a matching CLI/app build.",
		protocolVersionLabel(e.Decision.DaemonProtocolVersion),
		e.Decision.ClientProtocolVersion,
		formatProtocolVersions(e.Decision.SupportedDaemonProtocolVersions),
		formatProtocolVersions(e.Decision.DaemonSupportedPreviousProtocolVersions),
	)
}

func supportedProtocolVersions(current int, supported []int) []int {
	if current == 0 {
		current = ProtocolVersion
	}
	versions := make([]int, 0, 1+len(supported))
	seen := map[int]bool{}
	for _, version := range append([]int{current}, supported...) {
		if version == 0 || seen[version] {
			continue
		}
		versions = append(versions, version)
		seen[version] = true
	}
	return versions
}

func containsProtocolVersion(versions []int, target int) bool {
	for _, version := range versions {
		if version == target {
			return true
		}
	}
	return false
}

func protocolVersionLabel(version int) string {
	if version == 0 {
		return "unknown"
	}
	return strconv.Itoa(version)
}

func formatProtocolVersions(versions []int) string {
	if len(versions) == 0 {
		return "none"
	}
	labels := make([]string, 0, len(versions))
	for _, version := range versions {
		labels = append(labels, protocolVersionLabel(version))
	}
	return strings.Join(labels, ", ")
}
