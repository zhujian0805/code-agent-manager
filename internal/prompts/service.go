package prompts

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// claudeLibraryJSON holds the bundled Claude Code prompt library, mirrored from
// https://code.claude.com/docs/en/prompt-library.  That page renders its prompt
// cards client-side, so the list cannot be scraped with a plain HTTP client;
// the prompts are embedded into the binary instead.
//
//go:embed embed/claude_library.json
var claudeLibraryJSON []byte

// Service handles fetching and syncing prompts from external sources.
type Service struct {
	store  *Store
	client *http.Client
}

// NewService creates a new prompts service.
func NewService() *Service {
	return &Service{
		store: NewStore(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Store returns the underlying store.
func (s *Service) Store() *Store {
	return s.store
}

// SyncAll fetches prompts from all configured sources.
func (s *Service) SyncAll(ctx context.Context) (added int, err error) {
	n0, e0 := s.syncClaudeLibrary(ctx)
	if e0 != nil {
		err = e0
	}
	added += n0

	n1, e1 := s.syncPromptsChat(ctx)
	if e1 != nil {
		err = e1
	}
	added += n1

	n2, e2 := s.syncPromptingGuide(ctx)
	if e2 != nil {
		err = e2
	}
	added += n2
	return added, err
}

// syncPromptsChat fetches prompts from f/prompts.chat GitHub repo (prompts.csv).
func (s *Service) syncPromptsChat(ctx context.Context) (int, error) {
	url := "https://raw.githubusercontent.com/f/prompts.chat/main/prompts.csv"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("prompts.chat: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prompts.chat: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("prompts.chat: parse csv: %w", err)
	}

	var count int
	for i, record := range records {
		if i == 0 {
			continue // skip header
		}
		if len(record) < 4 {
			continue
		}
		// CSV columns: act, prompt, description, tags
		act := strings.TrimSpace(record[0])
		prompt := strings.TrimSpace(record[1])
		description := strings.TrimSpace(record[2])
		tags := strings.TrimSpace(record[3])

		if act == "" || prompt == "" {
			continue
		}

		p := &Prompt{
			Source:      "prompts_chat",
			SourceURL:   fmt.Sprintf("https://prompts.chat/prompts#%s", strings.ToLower(strings.ReplaceAll(act, " ", "-"))),
			Category:    categorizePrompt(tags, description),
			Title:       act,
			Description: description,
			Content:     prompt,
			Author:      "community",
			Tags:        tags,
		}
		if err := s.store.UpsertPrompt(ctx, p); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

// syncPromptingGuide fetches prompts from dair-ai/Prompt-Engineering-Guide.
// It walks every example-bearing guide file and extracts each prompt block.
func (s *Service) syncPromptingGuide(ctx context.Context) (int, error) {
	// The guide splits its examples across several markdown files.  Fetching only
	// one would miss most of the prompts, so pull every example-bearing page.
	files := []string{
		"prompts-intro.md",
		"prompts-basic-usage.md",
		"prompts-advanced-usage.md",
		"prompts-applications.md",
		"prompts-chatgpt.md",
		"prompts-miscellaneous.md",
	}

	var count int
	var firstErr error
	seen := make(map[string]bool)
	for _, file := range files {
		url := "https://raw.githubusercontent.com/dair-ai/Prompt-Engineering-Guide/main/guides/" + file
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		resp, err := s.client.Do(req)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("promptingguide: fetch %s: %w", file, err)
			}
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if firstErr == nil {
				firstErr = fmt.Errorf("promptingguide: %s status %d", file, resp.StatusCode)
			}
			continue
		}
		if readErr != nil {
			if firstErr == nil {
				firstErr = readErr
			}
			continue
		}

		for _, p := range parsePromptsFromMarkdown(string(body)) {
			// Give every prompt a unique source_url so they don't collide on the
			// (source, source_url) unique index.  De-dupe identical titles that
			// appear in more than one guide file.
			slug := slugify(p.Title)
			if slug == "" || seen[slug] {
				continue
			}
			seen[slug] = true
			p.Source = "promptingguide"
			p.SourceURL = "https://www.promptingguide.ai/introduction/examples#" + slug
			p.Author = "DAIR.AI"
			if err := s.store.UpsertPrompt(ctx, p); err != nil {
				continue
			}
			count++
		}
	}
	return count, firstErr
}

// parsePromptsFromMarkdown extracts prompts from markdown content.
func parsePromptsFromMarkdown(content string) []*Prompt {
	var prompts []*Prompt

	// Split by headers
	sections := regexp.MustCompile(`(?m)^## `).Split(content, -1)

	for _, section := range sections {
		lines := strings.Split(section, "\n")
		if len(lines) == 0 {
			continue
		}

		title := strings.TrimSpace(lines[0])
		if title == "" || strings.HasPrefix(title, "#") {
			continue
		}

		// Look for prompt blocks (indented text or code blocks)
		var promptLines []string
		inPrompt := false
		for _, line := range lines[1:] {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Prompt:") || strings.HasPrefix(trimmed, "*Prompt:*") {
				inPrompt = true
				continue
			}
			if inPrompt {
				// Skip the markdown code fences that wrap the prompt body.
				if strings.HasPrefix(trimmed, "```") {
					if len(promptLines) > 0 {
						break // closing fence after content was collected
					}
					continue // opening fence
				}
				if trimmed == "" || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "Here") {
					if len(promptLines) > 0 {
						break
					}
				} else {
					promptLines = append(promptLines, line)
				}
			}
		}

		content := strings.TrimSpace(strings.Join(promptLines, "\n"))
		if content != "" {
			prompts = append(prompts, &Prompt{
				Category:    categorizePrompt("", title),
				Title:       title,
				Description: title,
				Content:     content,
			})
		}
	}
	return prompts
}

// slugify converts a prompt title into a URL-fragment-safe slug so each prompt
// gets a distinct source_url (the store enforces a unique (source, source_url)
// index, so colliding URLs would silently drop all but one prompt).
var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// categorizePrompt determines the category based on tags and description.
func categorizePrompt(tags, description string) string {
	combined := strings.ToLower(tags + " " + description)

	switch {
	case strings.Contains(combined, "code") || strings.Contains(combined, "programming") || strings.Contains(combined, "developer"):
		return "coding"
	case strings.Contains(combined, "write") || strings.Contains(combined, "creative") || strings.Contains(combined, "story"):
		return "writing"
	case strings.Contains(combined, "analyz") || strings.Contains(combined, "research") || strings.Contains(combined, "summariz"):
		return "analysis"
	case strings.Contains(combined, "learn") || strings.Contains(combined, "teach") || strings.Contains(combined, "explain"):
		return "education"
	case strings.Contains(combined, "business") || strings.Contains(combined, "market") || strings.Contains(combined, "plan"):
		return "business"
	case strings.Contains(combined, "image") || strings.Contains(combined, "photo") || strings.Contains(combined, "art"):
		return "creative"
	case strings.Contains(combined, "math") || strings.Contains(combined, "logic") || strings.Contains(combined, "reason"):
		return "reasoning"
	default:
		return "general"
	}
}

// ClaudePrompt represents a single prompt in the bundled Claude Code prompt library.
type ClaudePrompt struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Tags     string `json:"tags"`
	Content  string `json:"content"`
}

// claudeLibrary is the on-disk shape of embed/claude_library.json.
type claudeLibrary struct {
	Prompts []ClaudePrompt `json:"prompts"`
}

// FetchClaudePrompts returns the prompts bundled from the Claude Code prompt
// library (https://code.claude.com/docs/en/prompt-library).  The library page is
// rendered client-side, so the prompts are embedded in the binary rather than
// fetched over HTTP; this keeps the source fully offline-capable.
func (s *Service) FetchClaudePrompts(ctx context.Context) ([]*ClaudePrompt, error) {
	var lib claudeLibrary
	if err := json.Unmarshal(claudeLibraryJSON, &lib); err != nil {
		return nil, fmt.Errorf("claude: parse bundled library: %w", err)
	}
	out := make([]*ClaudePrompt, 0, len(lib.Prompts))
	for i := range lib.Prompts {
		out = append(out, &lib.Prompts[i])
	}
	return out, nil
}

// syncClaudeLibrary loads the bundled Claude prompt library into the store.
func (s *Service) syncClaudeLibrary(ctx context.Context) (int, error) {
	prompts, err := s.FetchClaudePrompts(ctx)
	if err != nil {
		return 0, err
	}
	var count int
	for _, cp := range prompts {
		if cp.Title == "" || cp.Content == "" {
			continue
		}
		p := &Prompt{
			Source:      "claude",
			SourceURL:   "https://code.claude.com/docs/en/prompt-library#" + slugify(cp.Title),
			Category:    cp.Category,
			Title:       cp.Title,
			Description: cp.Content,
			Content:     cp.Content,
			Author:      "Anthropic",
			Tags:        cp.Tags,
		}
		if err := s.store.UpsertPrompt(ctx, p); err != nil {
			continue
		}
		count++
	}
	return count, nil
}

// RefreshSource syncs a specific source.
func (s *Service) RefreshSource(ctx context.Context, source string) (int, error) {
	switch source {
	case "prompts_chat":
		return s.syncPromptsChat(ctx)
	case "promptingguide":
		return s.syncPromptingGuide(ctx)
	case "claude":
		return s.syncClaudeLibrary(ctx)
	default:
		return 0, fmt.Errorf("unknown source: %s", source)
	}
}

// SourceStatus returns the status of each prompt source.
type SourceStatus struct {
	Source      string `json:"source"`
	Name        string `json:"name"`
	LastSync    string `json:"last_sync"`
	PromptCount int    `json:"prompt_count"`
	Enabled     bool   `json:"enabled"`
}

// GetSourceStatus returns the status of all prompt sources.
func (s *Service) GetSourceStatus(ctx context.Context) ([]SourceStatus, error) {
	sources := []string{"claude", "prompts_chat", "promptingguide"}
	var statuses []SourceStatus

	for _, source := range sources {
		count, _ := s.store.CountPrompts(ctx, source, "")
		statuses = append(statuses, SourceStatus{
			Source:      source,
			Name:        sourceDisplayName(source),
			PromptCount: count,
			Enabled:     true,
		})
	}
	return statuses, nil
}

func sourceDisplayName(source string) string {
	switch source {
	case "claude":
		return "Claude Prompt Library"
	case "prompts_chat":
		return "prompts.chat"
	case "promptingguide":
		return "Prompt Engineering Guide"
	default:
		return source
	}
}
