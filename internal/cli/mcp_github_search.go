package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// searchGitHubMCP searches GitHub for MCP server repositories matching a query.
func searchGitHubMCP(query string, limit int) ([]ghSearchResult, error) {
	q := fmt.Sprintf("mcp-server %s in:name,description,readme", query)
	client := &http.Client{Timeout: 15 * time.Second}
	authHeader := resolveGitHubAuth()
	apiURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=%d&sort=stars&order=desc",
		url.QueryEscape(q), limit)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "code-agent-manager")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded, set GITHUB_TOKEN for higher limits")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			FullName        string `json:"full_name"`
			Description     string `json:"description"`
			StargazersCount int    `json:"stargazers_count"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	out := make([]ghSearchResult, 0, len(result.Items))
	for _, item := range result.Items {
		parts := strings.SplitN(item.FullName, "/", 2)
		name := item.FullName
		if len(parts) == 2 {
			name = parts[1]
		}
		out = append(out, ghSearchResult{
			Repo:        item.FullName,
			Name:        name,
			ID:          name,
			Description: item.Description,
			Stars:       item.StargazersCount,
		})
	}
	return out, nil
}
