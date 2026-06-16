package desktop

import (
	"bytes"

	"github.com/chat2anyllm/code-agent-manager/internal/tools"
)

type ToolService struct{}

func NewToolService() *ToolService { return &ToolService{} }

func (s *ToolService) List() ([]ToolDTO, error) {
	registry, err := tools.LoadDefault()
	if err != nil {
		return nil, wrapError("TOOL_REGISTRY_LOAD_FAILED", err)
	}
	out := make([]ToolDTO, 0, len(registry.Tools))
	for _, name := range registry.Names() {
		tool := registry.Tools[name]
		out = append(out, toolDTO(tool))
	}
	return out, nil
}

func (s *ToolService) Install(name string, dryRun bool) (OperationResult, error) {
	tool, err := loadTool(name)
	if err != nil {
		return OperationResult{}, err
	}
	if dryRun {
		return OperationResult{OK: true, Message: tool.InstallCmd}, nil
	}
	var stdout, stderr bytes.Buffer
	code, err := tools.Install(tool, nil, &stdout, &stderr)
	if err != nil {
		return OperationResult{}, wrapError("TOOL_INSTALL_FAILED", err)
	}
	return OperationResult{OK: code == 0, Message: stdout.String() + stderr.String()}, nil
}

func (s *ToolService) Uninstall(name string, dryRun bool) (OperationResult, error) {
	tool, err := loadTool(name)
	if err != nil {
		return OperationResult{}, err
	}
	if dryRun {
		return OperationResult{OK: true, Message: "uninstall " + tool.LaunchCommand()}, nil
	}
	var stdout, stderr bytes.Buffer
	code, msg, err := tools.Uninstall(tool, nil, &stdout, &stderr)
	if err != nil {
		return OperationResult{}, wrapError("TOOL_UNINSTALL_FAILED", err)
	}
	return OperationResult{OK: code == 0, Message: msg + "\n" + stdout.String() + stderr.String()}, nil
}

func (s *ToolService) Upgrade(name string, dryRun bool) (OperationResult, error) {
	return s.Install(name, dryRun)
}

func loadTool(name string) (tools.Tool, error) {
	registry, err := tools.LoadDefault()
	if err != nil {
		return tools.Tool{}, wrapError("TOOL_REGISTRY_LOAD_FAILED", err)
	}
	tool, ok := registry.Get(name)
	if !ok {
		if byCommand, found := registry.ByCLICommand(name); found {
			return byCommand, nil
		}
		return tools.Tool{}, NewError("TOOL_NOT_FOUND", "tool not found", map[string]string{"name": name})
	}
	return tool, nil
}

func toolDTO(tool tools.Tool) ToolDTO {
	return ToolDTO{
		Name: tool.Name, Command: tool.LaunchCommand(), Description: tool.Description,
		Enabled: tool.IsEnabled(), Installed: tools.IsInstalled(tool), Version: tools.DetectVersion(tool),
	}
}
