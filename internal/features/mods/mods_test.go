package mods

import "testing"

func testCategories() []Category {
	return []Category{
		{
			ID:          "turn_speed",
			Description: "turn speed",
			Modes: []Mode{
				{ID: "turn_speed_1_5x", Label: "1.5x"},
				{ID: "turn_speed_2x", Label: "2x"},
			},
		},
		{
			ID:          "full_analog_y_axis",
			Description: "y axis",
			Modes: []Mode{
				{ID: "y_inverted", Label: "Inverted", Requires: []string{"full_analog_controls"}},
			},
		},
		{
			ID:          "full_analog_controls",
			Description: "analog",
			Modes: []Mode{
				{ID: "full_analog_controls", Label: "Full analog"},
			},
		},
	}
}

func TestResolve_ValidSelectionPassesThrough(t *testing.T) {
	sel := Selection{{"turn_speed_2x": true}}
	resolved, err := Resolve(testCategories(), sel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved) != 1 || resolved[0].ID != "turn_speed_2x" {
		t.Errorf("resolved = %+v, want just turn_speed_2x", resolved)
	}
}

func TestResolve_MutualExclusion(t *testing.T) {
	sel := Selection{{"turn_speed_1_5x": true}, {"turn_speed_2x": true}}
	_, err := Resolve(testCategories(), sel)
	if err == nil {
		t.Fatal("expected an error when two modes in the same category are both enabled")
	}
}

func TestResolve_UnmetRequires(t *testing.T) {
	sel := Selection{{"y_inverted": true}}
	_, err := Resolve(testCategories(), sel)
	if err == nil {
		t.Fatal("expected an error when a mode's Requires isn't also enabled")
	}
}

func TestResolve_RequiresSatisfied(t *testing.T) {
	sel := Selection{{"y_inverted": true}, {"full_analog_controls": true}}
	resolved, err := Resolve(testCategories(), sel)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolved) != 2 {
		t.Errorf("resolved = %+v, want both modes", resolved)
	}
}

func TestResolve_UnknownModeID(t *testing.T) {
	sel := Selection{{"not_a_real_mode": true}}
	_, err := Resolve(testCategories(), sel)
	if err == nil {
		t.Fatal("expected an error for an unknown mode ID")
	}
}

func TestResolve_DuplicateModeID(t *testing.T) {
	sel := Selection{{"turn_speed_2x": true}, {"turn_speed_2x": false}}
	_, err := Resolve(testCategories(), sel)
	if err == nil {
		t.Fatal("expected an error when a mode ID is listed more than once")
	}
}
