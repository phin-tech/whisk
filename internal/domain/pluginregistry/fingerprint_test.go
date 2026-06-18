package pluginregistry_test

import (
	"testing"

	"github.com/phin-tech/whisk/internal/domain/pluginregistry"
)

func TestFingerprintStableAcrossOrder(t *testing.T) {
	a := map[string][]byte{"plugin.json": []byte(`{"id":"x"}`), "resolve.mjs": []byte("export {}")}
	b := map[string][]byte{"resolve.mjs": []byte("export {}"), "plugin.json": []byte(`{"id":"x"}`)}
	if pluginregistry.Fingerprint(a) != pluginregistry.Fingerprint(b) {
		t.Fatal("fingerprint changed with map order")
	}
}

func TestFingerprintDetectsContentChange(t *testing.T) {
	base := map[string][]byte{"plugin.json": []byte(`{"id":"x"}`)}
	changed := map[string][]byte{"plugin.json": []byte(`{"id":"y"}`)}
	if pluginregistry.Fingerprint(base) == pluginregistry.Fingerprint(changed) {
		t.Fatal("fingerprint ignored content change")
	}
}

func TestFingerprintNoBoundaryCollision(t *testing.T) {
	// Moving a byte across the path/content boundary must change the hash.
	left := map[string][]byte{"ab": []byte("c")}
	right := map[string][]byte{"a": []byte("bc")}
	if pluginregistry.Fingerprint(left) == pluginregistry.Fingerprint(right) {
		t.Fatal("fingerprint collided across path/content boundary")
	}
}
