package desktop

import (
	"sort"

	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

type MCPService struct{}

func NewMCPService() *MCPService { return &MCPService{} }

func (s *MCPService) ListClients() []MCPClientDTO {
	out := make([]MCPClientDTO, 0, len(mcp.SupportedClients))
	for _, client := range mcp.SupportedClients {
		out = append(out, MCPClientDTO{
			Name:         client.Name,
			UserPath:     client.UserPath,
			ProjectPath:  client.ProjectPath,
			Container:    client.Container,
			Format:       client.Format,
			SupportsUser: client.UserPath != "",
			SupportsProj: client.ProjectPath != "",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *MCPService) ListInstalled(clientName, scope string) ([]MCPServerDTO, error) {
	client, ok := mcp.ClientByName(clientName)
	if !ok {
		return nil, NewError("MCP_CLIENT_NOT_FOUND", "unsupported MCP client", map[string]string{"client": clientName})
	}
	servers, path, err := mcp.ListServers(client, mcp.Scope(scope))
	if err != nil {
		return nil, wrapError("MCP_LIST_FAILED", err)
	}
	out := make([]MCPServerDTO, 0, len(servers))
	for _, server := range servers {
		out = append(out, mcpServerDTO(server, clientName, scope, path))
	}
	return out, nil
}

func (s *MCPService) Add(clientName, scope string, input MCPServerDTO) (OperationResult, error) {
	client, ok := mcp.ClientByName(clientName)
	if !ok {
		return OperationResult{}, NewError("MCP_CLIENT_NOT_FOUND", "unsupported MCP client", map[string]string{"client": clientName})
	}
	path, err := mcp.AddServer(client, mcp.Scope(scope), mcp.Server{
		Name: input.Name, Command: input.Command, Args: input.Args, Env: input.Env, URL: input.URL, Type: input.Type,
	})
	if err != nil {
		return OperationResult{}, wrapError("MCP_ADD_FAILED", err)
	}
	return OperationResult{OK: true, Message: "MCP server added", Path: path}, nil
}

func (s *MCPService) Remove(clientName, scope, name string) (OperationResult, error) {
	client, ok := mcp.ClientByName(clientName)
	if !ok {
		return OperationResult{}, NewError("MCP_CLIENT_NOT_FOUND", "unsupported MCP client", map[string]string{"client": clientName})
	}
	path, removed, err := mcp.RemoveServer(client, mcp.Scope(scope), name)
	if err != nil {
		return OperationResult{}, wrapError("MCP_REMOVE_FAILED", err)
	}
	if !removed {
		return OperationResult{}, NewError("MCP_SERVER_NOT_FOUND", "MCP server not found", map[string]string{"name": name})
	}
	return OperationResult{OK: true, Message: "MCP server removed", Path: path}, nil
}

func (s *MCPService) SearchRegistry(query string) ([]mcp.ServerSchema, error) {
	registry, err := mcp.LoadBundledRegistry()
	if err != nil {
		return nil, wrapError("MCP_REGISTRY_LOAD_FAILED", err)
	}
	return registry.Search(query), nil
}

func (s *MCPService) ShowServer(name string) (mcp.ServerSchema, error) {
	registry, err := mcp.LoadBundledRegistry()
	if err != nil {
		return mcp.ServerSchema{}, wrapError("MCP_REGISTRY_LOAD_FAILED", err)
	}
	schema, ok := registry.Get(name)
	if !ok {
		return mcp.ServerSchema{}, NewError("MCP_SERVER_SCHEMA_NOT_FOUND", "MCP registry server not found", map[string]string{"name": name})
	}
	return schema, nil
}

func mcpServerDTO(server mcp.Server, client, scope, path string) MCPServerDTO {
	return MCPServerDTO{
		Name: server.Name, Client: client, Scope: scope, Path: path, Command: server.Command,
		Args: server.Args, Env: server.Env, URL: server.URL, Type: server.Type,
	}
}
