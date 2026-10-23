package main

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"kf4randomizer/internal/tracker"
)

//go:embed icons
var iconFS embed.FS

// trackerState is the one file the page may write, set only by /tracker/load.
type trackerState struct {
	mu   sync.Mutex
	last string // tracker written by the latest successful build
	path string
	file *tracker.File
}

var trk trackerState

func (s *trackerState) setLast(path string) {
	s.mu.Lock()
	s.last = path
	s.mu.Unlock()
}

func trackerTilesJSON() template.JS {
	raw, err := json.Marshal(tracker.Tiles)
	if err != nil {
		panic(err)
	}
	return template.JS(raw)
}

func registerTrackerHandlers(mux *http.ServeMux) {
	icons, err := fs.Sub(iconFS, "icons")
	if err != nil {
		panic(err)
	}
	mux.Handle("/icons/", http.StripPrefix("/icons/", http.FileServerFS(icons)))
	mux.HandleFunc("/tracker/last", handleTrackerLast)
	mux.HandleFunc("/tracker/load", handleTrackerLoad)
	mux.HandleFunc("/tracker/save", handleTrackerSave)
	mux.HandleFunc("/browse-tracker", handleBrowseTracker)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleTrackerLast(w http.ResponseWriter, _ *http.Request) {
	trk.mu.Lock()
	defer trk.mu.Unlock()
	writeJSON(w, map[string]string{"path": trk.last})
}

func handleTrackerLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil || req.Path == "" {
		http.Error(w, "body must be {\"path\": \"...\"}", http.StatusBadRequest)
		return
	}
	f, err := tracker.Load(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trk.mu.Lock()
	trk.path, trk.file = req.Path, f
	trk.mu.Unlock()

	caps := make(map[string]int, len(tracker.Tiles))
	for _, t := range tracker.Tiles {
		caps[t.ID] = f.Cap(t.ID)
	}
	writeJSON(w, struct {
		Path      string         `json:"path"`
		Seed      string         `json:"seed"`
		Generated time.Time      `json:"generated"`
		Max       map[string]int `json:"max"`
		State     map[string]int `json:"state"`
	}{req.Path, f.Seed, f.Generated, caps, f.State})
}

func handleTrackerSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		State map[string]int `json:"state"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "parsing state: "+err.Error(), http.StatusBadRequest)
		return
	}
	trk.mu.Lock()
	defer trk.mu.Unlock()
	if trk.file == nil {
		http.Error(w, "no tracker file is open", http.StatusConflict)
		return
	}
	next := *trk.file
	next.State = req.State
	if err := next.Save(trk.path); err != nil {
		http.Error(w, "saving tracker: "+err.Error(), http.StatusInternalServerError)
		return
	}
	trk.file = &next
	w.WriteHeader(http.StatusNoContent)
}

// handleBrowseTracker returns the picked tracker file as JSON, "" if cancelled.
func handleBrowseTracker(w http.ResponseWriter, _ *http.Request) {
	path, err := browseForFile("Select a tracker file", jsonFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"path": path})
}
