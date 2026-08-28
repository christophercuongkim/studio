package webui

import (
	"os/exec"
	"testing"
)

// TestSeaKimConformance runs the vendored SeaKim conformance checker against the
// embedded frontends (ADR 0012: the checks ship with the rules, the consumer
// runs them). Skips when node is absent so hermetic Go runs still pass; the
// devShell provides node.
func TestSeaKimConformance(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH (design conformance check skipped)")
	}
	const checker = "seakim/tool/conformance-check.mjs"
	for _, dir := range []string{"serve", "prompt"} {
		out, err := exec.Command("node", checker, dir).CombinedOutput()
		if err != nil {
			t.Errorf("SeaKim conformance failed for %s:\n%s", dir, out)
		}
	}
}
