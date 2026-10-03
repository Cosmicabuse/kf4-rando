package kfmap

import "testing"

func TestLoadMapNames(t *testing.T) {
	csv := "game_name,human_name\n000.map,Village Ruins\n006.map,\"King's Throne, upper\"\n"

	names, err := LoadMapNames([]byte(csv))
	if err != nil {
		t.Fatalf("LoadMapNames: %v", err)
	}

	if got, want := names["000.map"], "Village Ruins"; got != want {
		t.Errorf("000.map: got %q, want %q", got, want)
	}
	if got, want := names["006.map"], "King's Throne, upper"; got != want {
		t.Errorf("006.map: got %q, want %q (comma-in-value quoting)", got, want)
	}
	if _, ok := names["game_name"]; ok {
		t.Error("header row should not appear in the result")
	}

	if _, err := LoadMapNames([]byte("game_name,human_name\n000.map,Too,Many,Columns\n")); err == nil {
		t.Error("a row with the wrong number of columns should fail")
	}
}

func TestMapNames_Header(t *testing.T) {
	names := MapNames{"017.map": "The Forge (ice)"}

	if got, want := names.Header("017.map"), "017.map (The Forge (ice))"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := names.Header("999.map"), "999.map"; got != want {
		t.Errorf("unknown map: got %q, want %q (unchanged)", got, want)
	}

	var nilNames MapNames
	if got, want := nilNames.Header("000.map"), "000.map"; got != want {
		t.Errorf("nil MapNames: got %q, want %q", got, want)
	}
}
