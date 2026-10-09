package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	sshconfig "github.com/kevinburke/ssh_config"
)

// includePlaceholder replaces Include directives before ssh_config parses a
// file. The library would otherwise read included files itself, without the
// Match handling below, so Stargate resolves includes on its own instead.
const includePlaceholder = "StargateInclude"

// document is one parsed SSH config file together with the files its Include
// directives resolved to.
type document struct {
	path     string
	config   *sshconfig.Config
	includes map[*sshconfig.KV][]*document
}

func loadDocument(path, home string, depth int) (*document, error) {
	if depth > maxIncludeDepth {
		return nil, fmt.Errorf("include SSH config %q: %w", path, sshconfig.ErrDepthExceeded)
	}

	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SSH config %q: %w", path, err)
	}
	content, err := os.ReadFile(canonicalPath)
	if err != nil {
		return nil, fmt.Errorf("read SSH config %q: %w", canonicalPath, err)
	}

	return parseDocument(canonicalPath, content, home, depth)
}

func parseDocument(path string, content []byte, home string, depth int) (*document, error) {
	prepared, includeArguments := prepareConfig(content)
	cfg, err := sshconfig.DecodeBytes(prepared)
	if err != nil {
		return nil, fmt.Errorf("parse SSH config %q: %w", path, err)
	}

	doc := &document{
		path:     path,
		config:   cfg,
		includes: make(map[*sshconfig.KV][]*document),
	}
	for _, block := range cfg.Hosts {
		for _, node := range block.Nodes {
			kv, ok := node.(*sshconfig.KV)
			if !ok || !strings.EqualFold(kv.Key, includePlaceholder) {
				continue
			}
			index, err := strconv.Atoi(kv.Value)
			if err != nil || index < 0 || index >= len(includeArguments) {
				return nil, fmt.Errorf("parse SSH config %q: invalid Include at line %d", path, kv.Pos().Line)
			}
			included, err := loadIncludes(includeArguments[index], home, depth)
			if err != nil {
				return nil, err
			}
			doc.includes[kv] = included
		}
	}

	return doc, nil
}

func loadIncludes(arguments, home string, depth int) ([]*document, error) {
	var included []*document
	seenMatches := make(map[string]struct{})
	for _, directive := range splitArguments(arguments) {
		pattern := resolveIncludePattern(directive, home)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("expand Include pattern %q: %w", directive, err)
		}
		for _, match := range matches {
			if _, exists := seenMatches[match]; exists {
				continue
			}
			seenMatches[match] = struct{}{}

			doc, err := loadDocument(match, home, depth+1)
			if err != nil {
				return nil, err
			}
			included = append(included, doc)
		}
	}
	return included, nil
}

// get returns the first value for key that applies to alias, following
// OpenSSH's first-value-wins rule across included files.
func (d *document) get(alias, key string) string {
	for _, block := range d.config.Hosts {
		if !block.Matches(alias) {
			continue
		}
		for _, node := range block.Nodes {
			kv, ok := node.(*sshconfig.KV)
			if !ok {
				continue
			}
			if included, isInclude := d.includes[kv]; isInclude {
				for _, doc := range included {
					if value := doc.get(alias, key); value != "" {
						return value
					}
				}
				continue
			}
			if strings.EqualFold(kv.Key, key) {
				return kv.Value
			}
		}
	}
	return ""
}

// prepareConfig blanks Match blocks that ssh_config cannot evaluate and
// replaces Include directives with numbered placeholders. Blanking keeps line
// numbers in parse errors stable. ssh_config rejects every Match criterion
// except "all" and "host", so a single "Match user" or "Match exec" block would
// otherwise make the whole config unusable. Skipped blocks are never evaluated,
// so Match exec commands are not run; OpenSSH still applies them on connect.
func prepareConfig(content []byte) ([]byte, []string) {
	lines := strings.Split(string(content), "\n")
	var includeArguments []string
	skipping := false
	for index, line := range lines {
		keyword, arguments := splitDirective(line)
		switch strings.ToLower(keyword) {
		case "host":
			skipping = false
		case "match":
			skipping = !supportedMatch(arguments)
		}
		if skipping {
			lines[index] = ""
			continue
		}
		if strings.EqualFold(keyword, "include") {
			lines[index] = fmt.Sprintf("%s %d", includePlaceholder, len(includeArguments))
			includeArguments = append(includeArguments, arguments)
		}
	}
	return []byte(strings.Join(lines, "\n")), includeArguments
}

// supportedMatch reports whether ssh_config evaluates the criteria like
// OpenSSH would: a lone "all" or a single "host" pattern list.
func supportedMatch(arguments string) bool {
	fields := splitArguments(arguments)
	switch len(fields) {
	case 1:
		return strings.EqualFold(fields[0], "all")
	case 2:
		return strings.EqualFold(fields[0], "host")
	default:
		return false
	}
}

func splitDirective(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", ""
	}
	end := strings.IndexFunc(trimmed, func(character rune) bool {
		return unicode.IsSpace(character) || character == '='
	})
	if end < 0 {
		return trimmed, ""
	}
	arguments := strings.TrimLeftFunc(trimmed[end:], unicode.IsSpace)
	arguments = strings.TrimPrefix(arguments, "=")
	return trimmed[:end], strings.TrimSpace(arguments)
}

// splitArguments splits like OpenSSH's argument parser: whitespace separates
// arguments, double quotes group them, and an unquoted # starts a comment.
func splitArguments(arguments string) []string {
	var (
		fields  []string
		current strings.Builder
		inField bool
		quoted  bool
	)
	for _, character := range arguments {
		switch {
		case character == '"':
			quoted = !quoted
			inField = true
		case !quoted && unicode.IsSpace(character):
			if inField {
				fields = append(fields, current.String())
				current.Reset()
				inField = false
			}
		case !quoted && !inField && character == '#':
			return fields
		default:
			current.WriteRune(character)
			inField = true
		}
	}
	if inField {
		fields = append(fields, current.String())
	}
	return fields
}
