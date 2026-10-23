package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kf4randomizer/internal/tracker"
)

func post(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	registerHandlers(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	return rec
}

func writeTracker(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "tracker_1.json")
	f := &tracker.File{Seed: "18446744073709551615", Generated: time.Unix(0, 0).UTC(), State: map[string]int{}}
	if err := f.Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSaveWithoutFile(t *testing.T) {
	trk = trackerState{}
	if rec := post(t, "/tracker/save", `{"state":{}}`); rec.Code != http.StatusConflict {
		t.Fatalf("code %d, want 409", rec.Code)
	}
}

func TestLoadThenSave(t *testing.T) {
	trk = trackerState{}
	p := writeTracker(t, t.TempDir())
	body, _ := json.Marshal(map[string]string{"path": p})
	rec := post(t, "/tracker/load", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("load: %d %s", rec.Code, rec.Body)
	}
	var got struct {
		Seed  string         `json:"seed"`
		State map[string]int `json:"state"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Seed != "18446744073709551615" {
		t.Fatalf("load body %s (%v)", rec.Body, err)
	}
	if rec := post(t, "/tracker/save", `{"state":{"keystone":2}}`); rec.Code != http.StatusNoContent {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	f, err := tracker.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.State["keystone"] != 2 || f.Seed != "18446744073709551615" {
		t.Fatalf("state %v seed %s", f.State["keystone"], f.Seed)
	}
}

func TestLoadRejectsOtherJSON(t *testing.T) {
	trk = trackerState{}
	p := filepath.Join(t.TempDir(), "settings.json")
	const raw = `{"editor.fontSize": 12}`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"path": p})
	if rec := post(t, "/tracker/load", string(body)); rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d, want 400", rec.Code)
	}
	if got, _ := os.ReadFile(p); string(got) != raw {
		t.Errorf("file changed: %s", got)
	}
}

func TestSaveFails(t *testing.T) {
	trk = trackerState{}
	dir := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := writeTracker(t, dir)
	body, _ := json.Marshal(map[string]string{"path": p})
	if rec := post(t, "/tracker/load", string(body)); rec.Code != http.StatusOK {
		t.Fatalf("load: %d", rec.Code)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	rec := post(t, "/tracker/save", `{"state":{}}`)
	if rec.Code != http.StatusInternalServerError || strings.TrimSpace(rec.Body.String()) == "" {
		t.Fatalf("code %d body %q", rec.Code, rec.Body)
	}
}

func TestIconsEmbedded(t *testing.T) {
	names := []string{}
	for _, tl := range tracker.Tiles {
		if tl.ID == "sword" {
			names = append(names, "sword_1", "sword_2")
			continue
		}
		names = append(names, tl.ID)
	}
	mux := http.NewServeMux()
	registerHandlers(mux)
	for _, n := range names {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/icons/"+n+".png", nil))
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Errorf("icons/%s.png: %d", n, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/icons/sword.png", nil))
	if rec.Code == http.StatusOK {
		t.Error("icons/sword.png should not exist")
	}
}
