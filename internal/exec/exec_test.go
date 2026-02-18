package exec

import "testing"

func TestCommandExists(t *testing.T) {
	if !CommandExists("go") {
		t.Error("CommandExists(\"go\") = false, want true")
	}
	if CommandExists("nonexistent-binary-xyz-123") {
		t.Error("CommandExists(\"nonexistent-binary-xyz-123\") = true, want false")
	}
}
