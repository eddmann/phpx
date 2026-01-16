package metadata

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/BurntSushi/toml"
)

// Metadata represents the parsed // phpx block from a PHP script.
type Metadata struct {
	PHP        string   `toml:"php"`
	Packages   []string `toml:"packages"`
	Extensions []string `toml:"extensions"`
}

// Parse extracts metadata from a PHP script's // phpx comment block.
//
// The block format is:
//
//	// phpx
//	// php = ">=8.2"
//	// packages = ["vendor/package:^1.0"]
//	// extensions = ["redis"]
func Parse(content []byte) (*Metadata, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Find the // phpx marker
	found := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "// phpx" {
			found = true
			break
		}
	}

	if !found {
		return &Metadata{}, nil
	}

	// Collect TOML lines
	var tomlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Stop at first non-comment line
		if !strings.HasPrefix(trimmed, "//") {
			break
		}

		// Strip the // prefix
		content := strings.TrimPrefix(trimmed, "//")
		// Remove leading space if present
		content = strings.TrimPrefix(content, " ")
		tomlLines = append(tomlLines, content)
	}

	if len(tomlLines) == 0 {
		return &Metadata{}, nil
	}

	// Parse as TOML
	tomlContent := strings.Join(tomlLines, "\n")
	var meta Metadata
	if _, err := toml.Decode(tomlContent, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
