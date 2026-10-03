package runner

import "testing"

func TestParseConfigRequiredFields(t *testing.T) {
	for _, c := range []struct {
		yaml    string
		wantErr bool
	}{
		{"output: out.iso\n", true},
		{"input: in.iso\n", true},
		{"input: in.iso\noutput: out.iso\n", false},
		// debug_data writes no ISO, so it needs no output.
		{"input: in.iso\ndebug_data: true\n", false},
	} {
		_, err := ParseConfig([]byte(c.yaml))
		if (err != nil) != c.wantErr {
			t.Errorf("ParseConfig(%q) error = %v, want error: %v", c.yaml, err, c.wantErr)
		}
	}
}

func TestParseConfigSeed(t *testing.T) {
	cfg, err := ParseConfig([]byte("input: in.iso\noutput: out.iso\n"))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Seed == nil {
		t.Error("an omitted seed should be filled in")
	}

	cfg, err = ParseConfig([]byte("input: in.iso\noutput: out.iso\nseed: 42\n"))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Seed == nil || *cfg.Seed != 42 {
		t.Errorf("Seed = %v, want 42", cfg.Seed)
	}

	cfg, err = ParseConfig([]byte("input: in.iso\ndebug_data: true\n"))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Seed != nil {
		t.Errorf("Seed = %d, want none; debug_data never shuffles", *cfg.Seed)
	}
}
