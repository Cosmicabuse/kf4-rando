package pnach

import "testing"

const sample = `// a comment
[1.5x turn speed]
patch=1,EE,0020D058,word,3C023F00

[Empty Section]

[Two Patches]
patch=1,EE,00210000,word,00000001
patch=1,EE,00210004,word,00000002
`

func TestParse(t *testing.T) {
	sections, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}

	if sections[0].Name != "1.5x turn speed" {
		t.Errorf("section 0 name = %q", sections[0].Name)
	}
	if len(sections[0].Patches) != 1 {
		t.Fatalf("section 0 has %d patches, want 1", len(sections[0].Patches))
	}
	p := sections[0].Patches[0]
	if p.Address != 0x0020D058 || p.Type != "word" || p.Value != 0x3C023F00 {
		t.Errorf("section 0 patch = %+v", p)
	}

	if sections[1].Name != "Empty Section" || len(sections[1].Patches) != 0 {
		t.Errorf("section 1 = %+v, want empty", sections[1])
	}

	if len(sections[2].Patches) != 2 {
		t.Fatalf("section 2 has %d patches, want 2", len(sections[2].Patches))
	}
}

func TestParse_PatchBeforeSection(t *testing.T) {
	_, err := Parse("patch=1,EE,00000000,word,00000000\n")
	if err == nil {
		t.Fatal("expected an error for a patch line before any [Section] header")
	}
}

func TestParse_MalformedPatchLine(t *testing.T) {
	_, err := Parse("[Section]\npatch=1,EE,not-hex,word,00000000\n")
	if err == nil {
		t.Fatal("expected an error for a non-hex address")
	}
}

func TestParse_UnclosedSectionHeader(t *testing.T) {
	_, err := Parse("[unterminated\n")
	if err == nil {
		t.Fatal("expected an error for a section header with no closing ]")
	}
}
