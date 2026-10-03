// Package runner is the build both commands drive. Progress goes to an
// io.Writer so the GUI can stream it.
package runner

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"kf4randomizer/internal/features/mods"
)

type Config struct {
	// Input is the source ISO. Always required.
	Input string `yaml:"input"`

	// Output is where the randomized ISO gets written. Required unless
	// DebugData is true.
	Output string `yaml:"output"`

	// Seed is the shuffle's PRNG seed. Omit for a fresh random one.
	Seed *uint64 `yaml:"seed"`

	// Spoiler overrides the spoiler log path. Defaults to
	// spoiler_<seed>.log next to Output.
	Spoiler string `yaml:"spoiler"`

	// DebugData switches to the debug-data subcommand: dumps raw item/map
	// data to debug_data/ and ignores every other field.
	DebugData bool `yaml:"debug_data"`

	// Mods selects which QoL patches to bake into Output.
	Mods mods.Selection `yaml:"mods"`

	// StartingConfig overrides ini/0611.ini fields by key, like
	// {"gold": 100}. Output's ini is always reset to vanilla plus these
	// overrides, regardless of Input's own ini.
	StartingConfig map[string]int `yaml:"starting_config"`

	// StripDummyFiles removes the 5 DUMMY*.OUT padding files, shrinking
	// Output by ~2.68GB.
	StripDummyFiles bool `yaml:"strip_dummy_files"`

	// RemoveEmptyContainers moves empty containers out of
	// every map.
	RemoveEmptyContainers bool `yaml:"remove_empty_containers"`

	// SharedItemModel repoints every map's first item model/texture pair
	// to one embedded model, so pickups look identical everywhere. Needs
	// the whitelist patch, which the randomizer always applies.
	SharedItemModel bool `yaml:"shared_item_model"`

	// SharedItemModelScale multiplies the shared model's size. 1 (or 0,
	// meaning unset) leaves it as authored.
	SharedItemModelScale float64 `yaml:"shared_item_model_scale"`

	// SharedItemModelColor flattens the shared model to one colour, as
	// "#rrggbb". Empty keeps the borrowed texture as-is.
	SharedItemModelColor string `yaml:"shared_item_model_color"`

	// RandomizeContainerItems folds container contents into the
	// ground-item shuffle pool.
	RandomizeContainerItems bool `yaml:"randomize_container_items"`

	// LogicFill makes every seed finishable. It shuffles containers too, so
	// RandomizeContainerItems doesn't apply. On unless set to false.
	LogicFill *bool `yaml:"logic_fill"`

	// PickupPopupShowItemIndex appends " (#N)" to the pickup popup, N
	// being the item's slot index (matches the spoiler log's numbering).
	PickupPopupShowItemIndex bool `yaml:"pickup_popup_show_item_index"`

	// FullDebugMode adds the L3+R3 menu: item HUD, map warp, levitate.
	FullDebugMode bool `yaml:"full_debug_mode"`

	// DisableRandomizer keeps ground items and containers
	// at vanilla placement regardless of the other randomize flags. QoL
	// settings (mods, RemoveEmptyContainers, starting_config,
	// EnableParamdomizer, ...) are unaffected.
	DisableRandomizer bool `yaml:"disable_randomizer"`

	// RandomizeMonsterDrops shuffles the drops set on placed monsters. Omit it
	// to leave it on.
	RandomizeMonsterDrops *bool `yaml:"randomize_monster_drops"`

	// RandomizeShops moves what the four gold shops sell between them.
	RandomizeShops bool `yaml:"randomize_shops"`

	// EnableParamdomizer randomizes weapon/armor stat values (not
	// placement).
	EnableParamdomizer bool `yaml:"enable_paramdomizer"`

	// ParamdomizerPercent is the max swing from a stat's original total,
	// as a percent each direction. Defaults to 25 if left at 0.
	ParamdomizerPercent int `yaml:"paramdomizer_percent"`

	// ParamdomizerAllowNegative lets individual weapon stats roll
	// negative. Armor never goes negative.
	ParamdomizerAllowNegative bool `yaml:"paramdomizer_allow_negative"`

	// ParamdomizerAllowWorseLevels lets a weapon's level-2/3 total roll
	// lower than the level before it. Default floors each level at the
	// previous one's total.
	ParamdomizerAllowWorseLevels bool `yaml:"paramdomizer_allow_worse_levels"`
}

// LoadConfig reads and validates a Config from a YAML file.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	return ParseConfig(raw)
}

// ParseConfig validates a Config from already-read YAML bytes.
func ParseConfig(raw []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Seed == nil && !cfg.DebugData {
		seed, err := RandomSeed()
		if err != nil {
			return nil, fmt.Errorf("generating a random seed: %w", err)
		}
		cfg.Seed = &seed
	}

	if cfg.EnableParamdomizer && cfg.ParamdomizerPercent == 0 {
		cfg.ParamdomizerPercent = 25
	}

	if cfg.Input == "" {
		return nil, fmt.Errorf("\"input\" is required")
	}
	if !cfg.DebugData && cfg.Output == "" {
		return nil, fmt.Errorf("\"output\" is required unless \"debug_data: true\" is set")
	}

	return &cfg, nil
}

// RandomSeed generates a fresh cryptographically random 64-bit seed.
func RandomSeed() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

// UseMonsterDrops reports whether monster drops are shuffled. Leaving the key
// out of the config means yes.
func (c *Config) UseMonsterDrops() bool {
	return c.RandomizeMonsterDrops == nil || *c.RandomizeMonsterDrops
}

// UseLogicFill reports whether to place items by reachability. Leaving the
// key out of the config means yes.
func (c *Config) UseLogicFill() bool {
	return c.LogicFill == nil || *c.LogicFill
}
