package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The MD5 check runs before ISO parsing, so a throwaway non-ISO file
// works as a fixture. Everything past it needs a real ISO, untested here.
func TestValidateInput_MD5Mismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-an-iso.iso")
	if err := os.WriteFile(path, []byte("definitely not a King's Field IV ISO"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	err := validateInput(path)
	if err == nil {
		t.Fatal("expected an error for a non-matching ISO, got nil")
	}
	if !strings.Contains(err.Error(), "input ISO MD5") {
		t.Errorf("error = %q, want it to mention \"MD5 mismatch\"", err.Error())
	}
}

func TestValidateInput_MissingFile(t *testing.T) {
	err := validateInput(filepath.Join(t.TempDir(), "does-not-exist.iso"))
	if err == nil {
		t.Fatal("expected an error for a missing input file, got nil")
	}
}
