package cli

import (
	"fmt"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

func clientsForList(name string) []mcp.ClientSpec {
	if name == "" {
		return mcp.SupportedClients
	}
	if c, ok := mcp.ClientByName(name); ok {
		return []mcp.ClientSpec{c}
	}
	return nil
}

func requireClient(name string) (mcp.ClientSpec, error) {
	if name == "" {
		return mcp.ClientSpec{}, fmt.Errorf("--client is required")
	}
	c, ok := mcp.ClientByName(name)
	if !ok {
		return mcp.ClientSpec{}, fmt.Errorf("unsupported client: %s (try one of: %s)", name, strings.Join(mcp.ClientNames(), ", "))
	}
	return c, nil
}

func parseEnv(entries []string) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, entry := range entries {
		if index := strings.IndexByte(entry, '='); index > 0 {
			out[entry[:index]] = entry[index+1:]
		}
	}
	return out
}
