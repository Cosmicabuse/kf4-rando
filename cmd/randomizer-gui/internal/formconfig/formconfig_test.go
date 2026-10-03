package formconfig

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
)

func newFormRequest(values url.Values) *http.Request {
	return &http.Request{Form: values, PostForm: values}
}

func TestBuildStartingConfig(t *testing.T) {
	// item_NNN indexes by classID directly, not classID-1.
	r := newFormRequest(url.Values{
		"hp":           {"999"},
		"equip_weapon": {"195"},
		"inv_item":     {"8", "11", "13"},
		"inv_qty":      {"80", "1", ""},
	})
	got, err := buildStartingConfig(r)
	if err != nil {
		t.Fatalf("buildStartingConfig: %v", err)
	}
	for key, want := range map[string]int{"hp": 999, "equip_weapon": 195, "item_008": 80, "item_011": 1} {
		if got[key] != want {
			t.Errorf("%s = %d, want %d", key, got[key], want)
		}
	}
	if _, ok := got["item_013"]; ok {
		t.Error("an item picked with no quantity typed should be ignored")
	}
}

func TestBuildStartingConfigRejectsBadInput(t *testing.T) {
	for why, values := range map[string]url.Values{
		"core stat out of range":    {"hp": {"9999999"}},
		"core stat not a number":    {"hp": {"abc"}},
		"inventory classID is text": {"inv_item": {"not-a-number"}, "inv_qty": {"1"}},
		"quantity above 99":         {"inv_item": {"8"}, "inv_qty": {"100"}},
	} {
		if _, err := buildStartingConfig(newFormRequest(values)); err == nil {
			t.Errorf("%s should be rejected", why)
		}
	}

	// A blank field means "leave it vanilla", not an error.
	got, err := buildStartingConfig(newFormRequest(url.Values{"hp": {""}}))
	if err != nil {
		t.Fatalf("a blank field should be skipped: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestResolveSeed(t *testing.T) {
	if _, err := resolveSeed(""); err != nil {
		t.Errorf("an empty seed should be generated, not rejected: %v", err)
	}
	if seed, err := resolveSeed("12345"); err != nil || seed != 12345 {
		t.Errorf("resolveSeed(\"12345\") = %d, %v", seed, err)
	}
	if _, err := resolveSeed("not-a-number"); err == nil {
		t.Error("a non-numeric seed should be rejected")
	}
}

func TestFromFormPaths(t *testing.T) {
	r := newFormRequest(url.Values{
		"input": {filepath.Join("C:", "iso", "Kings Field.iso")},
		"seed":  {"42"},
	})
	cfg, err := FromForm(r)
	if err != nil {
		t.Fatalf("FromForm: %v", err)
	}
	if want := filepath.Join("C:", "iso", "KF4_42.iso"); cfg.Output != want {
		t.Errorf("Output = %q, want %q", cfg.Output, want)
	}

	// The page writes <seed> into the paths before a seed exists.
	r = newFormRequest(url.Values{
		"input":   {`C:\games\kf4.iso`},
		"output":  {`C:\games\KF4_<seed>.iso`},
		"spoiler": {`C:\games\spoiler_<seed>.log`},
		"seed":    {"77"},
	})
	cfg, err = FromForm(r)
	if err != nil {
		t.Fatalf("FromForm: %v", err)
	}
	if cfg.Output != `C:\games\KF4_77.iso` {
		t.Errorf("Output = %q", cfg.Output)
	}
	if cfg.Spoiler != `C:\games\spoiler_77.log` {
		t.Errorf("Spoiler = %q", cfg.Spoiler)
	}
}

func TestFromForm(t *testing.T) {
	cfg, err := FromForm(newFormRequest(url.Values{
		"input": {"in.iso"}, "output": {"out.iso"}, "seed": {"999"}, "hp": {"999"},
	}))
	if err != nil {
		t.Fatalf("FromForm: %v", err)
	}
	if cfg.Seed == nil || *cfg.Seed != 999 {
		t.Errorf("Seed = %v, want 999", cfg.Seed)
	}
	// hidden cheat fields must be ignored
	if len(cfg.StartingConfig) != 0 {
		t.Errorf("StartingConfig = %v, want nothing while cheats are off", cfg.StartingConfig)
	}

	for _, c := range []struct {
		logic string
		want  bool
	}{{"", true}, {"glitchless", true}, {"none", false}} {
		cfg, err := FromForm(newFormRequest(url.Values{"input": {"in.iso"}, "logic": {c.logic}}))
		if err != nil {
			t.Fatalf("FromForm: %v", err)
		}
		if cfg.UseLogicFill() != c.want {
			t.Errorf("logic=%q gave UseLogicFill %v, want %v", c.logic, cfg.UseLogicFill(), c.want)
		}
	}

	if _, err := FromForm(newFormRequest(url.Values{"seed": {"1"}})); err == nil {
		t.Error("a missing input should be rejected")
	}
	if _, err := FromForm(newFormRequest(url.Values{
		"input": {"in.iso"}, "enable_cheats": {"on"}, "hp": {"not-a-number"},
	})); err == nil {
		t.Error("an invalid starting-config field should be rejected")
	}
}

func TestParseUploadedYAML(t *testing.T) {
	// The page gets this as JSON, where a uint64 seed would lose precision.
	out, err := ParseUploadedYAML([]byte("input: in.iso\nseed: 12345678901234567890\nstarting_config:\n  item_008: 80\n"))
	if err != nil {
		t.Fatalf("ParseUploadedYAML: %v", err)
	}
	decoded := asJSON(t, out)
	if seed, _ := decoded["seed"].(string); seed != "12345678901234567890" {
		t.Errorf("seed = %v (%T), want it as a string", decoded["seed"], decoded["seed"])
	}
	sc, ok := decoded["starting_config"].(map[string]any)
	if !ok {
		t.Fatalf("starting_config missing or wrong type: %v", decoded["starting_config"])
	}
	if sc["item_008"] != float64(80) {
		t.Errorf("item_008 = %v, want 80", sc["item_008"])
	}

	out, err = ParseUploadedYAML([]byte("input: in.iso\noutput: out.iso\n"))
	if err != nil {
		t.Fatalf("ParseUploadedYAML: %v", err)
	}
	if seed := asJSON(t, out)["seed"]; seed != nil {
		t.Errorf("seed = %v, want null; the file never set one", seed)
	}

	if _, err := ParseUploadedYAML([]byte("not: [valid: yaml")); err == nil {
		t.Error("malformed YAML should be rejected")
	}
}

func asJSON(t *testing.T, out any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	return decoded
}
