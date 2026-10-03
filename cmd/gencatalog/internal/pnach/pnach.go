// Package pnach reads the [Section] blocks and patch= lines of a .pnach file.
package pnach

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Patch is one `patch=1,EE,<address>,<type>,<value>` line.
type Patch struct {
	Address uint32
	Type    string // "word", "extended", "short", "byte", ...
	Value   uint32
}

// Section is one [Name] block and its patch lines.
type Section struct {
	Name    string
	Patches []Patch
}

// Parse returns the sections in file order, empty ones included.
func Parse(text string) ([]Section, error) {
	var sections []Section
	var current *Section

	scanner := bufio.NewScanner(strings.NewReader(text))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			// last ']', a name can contain one
			end := strings.LastIndex(line, "]")
			if end < 0 {
				return nil, fmt.Errorf("line %d: %q looks like a section header but has no closing ]", lineNo, line)
			}
			if current != nil {
				sections = append(sections, *current)
			}
			current = &Section{Name: line[1:end]}
			continue
		}

		if strings.HasPrefix(line, "patch=") {
			p, err := parsePatchLine(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			if current == nil {
				return nil, fmt.Errorf("line %d: patch line before any [Section] header", lineNo)
			}
			current.Patches = append(current.Patches, p)
			continue
		}

		return nil, fmt.Errorf("line %d: unrecognized line %q", lineNo, line)
	}
	if current != nil {
		sections = append(sections, *current)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sections, nil
}

// parsePatchLine parses `patch=1,EE,<address>,<type>,<value>`.
func parsePatchLine(line string) (Patch, error) {
	fields := strings.Split(line, ",")
	if len(fields) != 5 {
		return Patch{}, fmt.Errorf("expected 5 comma-separated fields, got %d: %q", len(fields), line)
	}
	// fields[0] is "patch=1", fields[1] is the CPU ("EE" always, here).
	addr, err := strconv.ParseUint(fields[2], 16, 32)
	if err != nil {
		return Patch{}, fmt.Errorf("bad address %q: %w", fields[2], err)
	}
	patchType := fields[3]
	value, err := strconv.ParseUint(fields[4], 16, 32)
	if err != nil {
		return Patch{}, fmt.Errorf("bad value %q: %w", fields[4], err)
	}
	return Patch{Address: uint32(addr), Type: patchType, Value: uint32(value)}, nil
}
