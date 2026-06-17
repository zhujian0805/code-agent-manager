package metadata

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

// DiscoverCatalogResources reads a generated markdown catalog file from a repo
// and turns each catalog table row into a resource. Generated awesome repos use
// these tables instead of concrete SKILL.md/plugin.json manifests.
func DiscoverCatalogResources(root, catalogFile string, kind entities.Kind) []DiscoveredResource {
	catalogFile = strings.TrimSpace(strings.Trim(catalogFile, "/"))
	if catalogFile == "" {
		return nil
	}
	path := filepath.Join(root, filepath.FromSlash(catalogFile))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseCatalogMarkdown(string(data), filepath.ToSlash(catalogFile), kind)
}

func inferCatalogFile(root string, kind entities.Kind) string {
	for _, candidate := range catalogCandidates(kind) {
		path := filepath.Join(root, filepath.FromSlash(candidate))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if len(parseCatalogMarkdown(string(data), candidate, kind)) > 0 {
			return candidate
		}
	}
	return ""
}

func catalogCandidates(kind entities.Kind) []string {
	plural := string(kind) + "s"
	upper := strings.ToUpper(plural)
	lower := strings.ToLower(plural)
	return []string{
		"FULL-" + upper + ".md",
		"FULL-" + lower + ".md",
		"FULL_" + upper + ".md",
		"FULL_" + lower + ".md",
		"FULL-" + string(kind) + ".md",
		"FULL_" + string(kind) + ".md",
		"README.md",
		"readme.md",
	}
}

func parseCatalogMarkdown(content, relPath string, kind entities.Kind) []DiscoveredResource {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	var out []DiscoveredResource
	seen := map[string]bool{}

	for i := 0; i < len(lines)-1; i++ {
		headerLine := strings.TrimSpace(lines[i])
		separatorLine := strings.TrimSpace(lines[i+1])
		if !looksLikeTableHeader(headerLine, separatorLine) {
			continue
		}

		header := splitTableRow(headerLine)
		nameIdx, descIdx, ok := catalogColumnIndexes(header, kind)
		if !ok {
			continue
		}

		for j := i + 2; j < len(lines); j++ {
			line := strings.TrimSpace(lines[j])
			if line == "" || !strings.Contains(line, "|") {
				break
			}
			cells := splitTableRow(line)
			if nameIdx >= len(cells) {
				continue
			}
			name, source := cleanCatalogNameCell(cells[nameIdx])
			if name == "" || strings.EqualFold(name, "name") {
				continue
			}
			key := strings.ToLower(catalogInstallKeyName(name, source))
			if seen[key] {
				continue
			}
			desc := ""
			if descIdx >= 0 && descIdx < len(cells) {
				desc = cleanCatalogCell(cells[descIdx])
			}
			seen[key] = true
			out = append(out, DiscoveredResource{
				Name:           name,
				Description:    desc,
				RelPath:        relPath,
				ManifestRel:    relPath,
				InstallKeyName: catalogInstallKeyName(name, source),
			})
		}
		i++
	}

	return out
}

func looksLikeTableHeader(header, separator string) bool {
	if !strings.Contains(header, "|") || !strings.Contains(separator, "|") {
		return false
	}
	for _, cell := range splitTableRow(separator) {
		trimmed := strings.TrimSpace(cell)
		if trimmed == "" {
			continue
		}
		trimmed = strings.Trim(trimmed, ":")
		if len(trimmed) >= 3 && strings.Trim(trimmed, "-") == "" {
			return true
		}
	}
	return false
}

func catalogColumnIndexes(header []string, kind entities.Kind) (nameIdx, descIdx int, ok bool) {
	nameIdx = -1
	descIdx = -1
	kindName := strings.ToLower(string(kind))
	for i, cell := range header {
		cleaned := strings.ToLower(cleanCatalogCell(cell))
		switch {
		case descIdx == -1 && strings.Contains(cleaned, "description"):
			descIdx = i
		case nameIdx == -1 && (strings.Contains(cleaned, kindName) || cleaned == "name"):
			nameIdx = i
		}
	}
	if nameIdx == -1 && len(header) > 0 {
		nameIdx = 0
	}
	if descIdx == -1 && len(header) > 1 {
		descIdx = 1
	}
	if nameIdx == descIdx {
		descIdx = -1
	}
	return nameIdx, descIdx, nameIdx >= 0
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

var (
	imageRe = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	linkRe  = regexp.MustCompile(`\[([^\]]+)\]\(([^)]*)\)`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	spaceRe = regexp.MustCompile(`\s+`)
)

func cleanCatalogCell(cell string) string {
	cell = strings.TrimSpace(cell)
	cell = imageRe.ReplaceAllString(cell, "")
	for {
		next := linkRe.ReplaceAllString(cell, "$1")
		if next == cell {
			break
		}
		cell = next
	}
	cell = strings.ReplaceAll(cell, "`", "")
	cell = tagRe.ReplaceAllString(cell, "")
	cell = strings.Trim(cell, "*_ ")
	cell = spaceRe.ReplaceAllString(cell, " ")
	return strings.TrimSpace(cell)
}

func cleanCatalogNameCell(cell string) (name, source string) {
	if match := linkRe.FindStringSubmatch(cell); len(match) == 3 {
		source = strings.TrimSpace(match[2])
	}
	return cleanCatalogCell(cell), source
}

func catalogInstallKeyName(name, source string) string {
	if source == "" {
		return name
	}
	owner, repo, ok := githubRepoFromURL(source)
	if !ok {
		return name
	}
	return owner + "/" + repo + ":" + name
}

func githubRepoFromURL(raw string) (owner, repo string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "www.")
	if !strings.HasPrefix(strings.ToLower(raw), "github.com/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(raw, "github.com/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	repo = strings.TrimSuffix(parts[1], ".git")
	return parts[0], repo, true
}
