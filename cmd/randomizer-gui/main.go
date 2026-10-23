// Command randomizer-gui serves the GUI on a random localhost port, opens it
// in the browser and streams the run's output back to the page.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"

	"gopkg.in/yaml.v3"

	"kf4randomizer/cmd/randomizer-gui/internal/formconfig"
	"kf4randomizer/internal/features/mods"
	"kf4randomizer/internal/runner"
	"kf4randomizer/internal/tracker"
)

func main() {
	noBrowser := flag.Bool("no-open", false, "don't automatically open a browser tab (the server still starts and prints its URL)")
	flag.Parse()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("starting local server: %v", err)
	}
	url := fmt.Sprintf("http://%s/", listener.Addr().String())

	mux := http.NewServeMux()
	registerHandlers(mux)

	fmt.Printf("KF4 Randomizer GUI running at %s\n", url)
	fmt.Println("(Leave this window open while using the GUI. Closing it stomps the server.)")

	if !*noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Printf("Couldn't open a browser automatically (%v). Open %s yourself.\n", err, url)
		}
	}

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

//go:embed index.html.tmpl
var templateFS embed.FS

var pageTemplate = template.Must(template.ParseFS(templateFS, "index.html.tmpl"))

type fileFilter struct{ desc, pattern string }

var (
	isoFilter  = fileFilter{"ISO files (*.iso)", "*.iso"}
	jsonFilter = fileFilter{"Tracker files (*.json)", "*.json"}
)

// one run at a time
var runMu sync.Mutex

func registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/config.yaml", handleDownloadConfig)
	mux.HandleFunc("/load-config", handleLoadConfig)
	mux.HandleFunc("/browse-input", handleBrowseInput)
	mux.HandleFunc("/browse-output", handleBrowseOutput)
	registerTrackerHandlers(mux)
}

type templateData struct {
	Categories      []mods.Category
	CoreStats       []formconfig.StatField
	MagicSpells     []formconfig.ElementSpells
	EquipSlots      []formconfig.EquipSlotView
	ItemCatalog     []formconfig.ItemCategoryOptions
	TrackerTiles    template.JS
	TrackerSections []string
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := templateData{
		Categories:      mods.BuiltinCategories,
		CoreStats:       formconfig.CoreStats,
		MagicSpells:     formconfig.MagicSpells,
		EquipSlots:      formconfig.EquipSlotViews,
		ItemCatalog:     formconfig.ItemCatalog,
		TrackerTiles:    trackerTilesJSON(),
		TrackerSections: tracker.Sections,
	}
	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg, err := formconfig.FromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !runMu.TryLock() {
		http.Error(w, "a run is already in progress, wait for it to finish", http.StatusConflict)
		return
	}
	defer runMu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, _ := w.(http.Flusher)
	fw := flushWriter{w: w, f: flusher}

	var runErr error
	var trackerPath string
	func() {
		defer func() {
			if p := recover(); p != nil {
				runErr = fmt.Errorf("internal error: %v", p)
			}
		}()
		trackerPath, runErr = runner.Run(fw, cfg)
	}()
	if runErr != nil {
		_, _ = fmt.Fprintf(fw, "\nERROR: %v\n", runErr)
	} else {
		trk.setLast(trackerPath)
		_, _ = fmt.Fprint(fw, "\n=== Done ===\n")
	}
}

// handleBrowseInput returns the picked source ISO as JSON, "" if cancelled.
func handleBrowseInput(w http.ResponseWriter, _ *http.Request) {
	path, err := browseForFile("Select the source ISO", isoFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"path": path})
}

// handleBrowseOutput returns the picked output path as JSON, "" if cancelled.
func handleBrowseOutput(w http.ResponseWriter, _ *http.Request) {
	path, err := browseForSavePath("Save the randomized ISO as")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"path": path})
}

// handleDownloadConfig serves a config.yaml built from the fields /run reads.
func handleDownloadConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := formconfig.FromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", `attachment; filename="config.yaml"`)
	_, _ = w.Write(out)
}

func handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB is plenty for a config
	if err != nil {
		http.Error(w, "reading upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	out, err := formconfig.ParseUploadedYAML(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
