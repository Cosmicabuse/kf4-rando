package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Seed      string         `json:"seed"` // string: a uint64 would be rounded by JavaScript
	Generated time.Time      `json:"generated"`
	Max       map[string]int `json:"max"` // per-seed caps overriding Tile.Max
	Start     map[string]int `json:"start"`
	State     map[string]int `json:"state"`
}

func tile(id string) *Tile {
	for i := range Tiles {
		if Tiles[i].ID == id {
			return &Tiles[i]
		}
	}
	return nil
}

// Cap is the highest state of a tile; never below 1 so a tile stays clickable.
func (f *File) Cap(id string) int {
	c, ok := f.Max[id]
	if !ok {
		if t := tile(id); t != nil {
			c = t.Max
		}
	}
	return max(c, 1)
}

// Load reads a tracker file and fits it to the current tile list.
func Load(path string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &File{}
	if err := json.Unmarshal(raw, f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.Seed == "" || (f.Start == nil && f.State == nil) {
		return nil, fmt.Errorf("%s: not a tracker file", path)
	}
	f.normalize()
	return f, nil
}

func (f *File) normalize() {
	start, state := map[string]int{}, map[string]int{}
	for _, t := range Tiles {
		start[t.ID] = min(max(f.Start[t.ID], 0), f.Cap(t.ID))
		v, ok := f.State[t.ID]
		if !ok {
			v = start[t.ID]
		}
		state[t.ID] = min(max(v, 0), f.Cap(t.ID))
	}
	f.Start, f.State = start, state
}

// Save writes through a temp file so a failed write keeps the old one.
func (f *File) Save(path string) error {
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tracker-*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(raw)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		// CreateTemp makes 0600 files
		err = os.Chmod(tmp.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// FreePath names a tracker file for seed without overwriting an existing one.
func FreePath(dir string, seed uint64) string {
	p := filepath.Join(dir, fmt.Sprintf("tracker_%d.json", seed))
	for n := 1; ; n++ {
		if _, err := os.Stat(p); err != nil {
			return p
		}
		p = filepath.Join(dir, fmt.Sprintf("tracker_%d-%d.json", seed, n))
	}
}
