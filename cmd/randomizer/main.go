// Command randomizer runs from config.yaml, next to the exe or passed as the
// only argument. -list_mods prints the QoL patch catalog.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"kf4randomizer/internal/features/mods"
	"kf4randomizer/internal/runner"
)

func main() {
	listMods := flag.Bool("list_mods", false, "print every available mod category/mode/ID and exit (no config needed)")
	flag.Parse()

	if *listMods {
		printModsCatalog()
		return
	}

	configPath := flag.Arg(0)
	if configPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("locating executable path: %v", err)
		}
		configPath = filepath.Join(filepath.Dir(exePath), "config.yaml")
	}

	cfg, err := runner.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.DebugData {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("locating executable path: %v", err)
		}
		outDir := filepath.Join(filepath.Dir(exePath), "debug_data")
		if err := runner.RunDebugData(os.Stdout, cfg.Input, outDir); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := runner.Run(os.Stdout, cfg); err != nil {
		log.Fatal(err)
	}
}

// printModsCatalog prints a mods: block to paste into config.yaml.
func printModsCatalog() {
	for _, cat := range mods.BuiltinCategories {
		fmt.Printf("%s: %s\n", cat.ID, cat.Description)
		for _, m := range cat.Modes {
			fmt.Printf("  %-40s %s\n", m.ID, m.Label)
			if m.Notes != "" {
				fmt.Printf("  %-40s   note: %s\n", "", m.Notes)
			}
		}
		fmt.Println()
	}
	fmt.Println("At most one mode per category may be set to true at once.")
	fmt.Println()
	fmt.Println("Paste into config.yaml, then flip the ones you want to true:")
	fmt.Println("mods:")
	for _, cat := range mods.BuiltinCategories {
		for _, m := range cat.Modes {
			fmt.Printf("  - %s: false\n", m.ID)
		}
	}
}
