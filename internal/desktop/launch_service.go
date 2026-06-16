package desktop

import (
	"os"
	"time"

	"github.com/chat2anyllm/code-agent-manager/internal/providers"
	"github.com/chat2anyllm/code-agent-manager/internal/tools"
)

type LaunchService struct {
	providersPath string
}

func NewLaunchService(providersPath string) *LaunchService {
	return &LaunchService{providersPath: providersPath}
}

func (s *LaunchService) ListTools() ([]ToolDTO, error) {
	return NewToolService().List()
}

func (s *LaunchService) ListProvidersForTool(toolName string) ([]ProviderDTO, error) {
	tool, err := loadTool(toolName)
	if err != nil {
		return nil, err
	}
	file, err := providers.Load(s.providersPath)
	if err != nil {
		return nil, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	out := []ProviderDTO{}
	for _, name := range file.SortedNames() {
		ep := file.Endpoints[name]
		if !ep.IsEnabled() {
			continue
		}
		if ep.SupportsClient(tool.LaunchCommand()) || ep.SupportsClient(tool.Name) || len(ep.Clients()) == 0 {
			out = append(out, providerDTO(name, ep))
		}
	}
	return out, nil
}

func (s *LaunchService) ListModelsForProvider(providerName string) ([]string, error) {
	file, err := providers.Load(s.providersPath)
	if err != nil {
		return nil, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	ep, ok := file.Endpoints[providerName]
	if !ok {
		return nil, NewError("PROVIDER_NOT_FOUND", "provider not found", map[string]string{"name": providerName})
	}
	models, err := providers.ResolveModels(ep, providerName, 24*time.Hour, "", os.Getenv)
	if err != nil {
		return nil, wrapError("MODEL_DISCOVERY_FAILED", err)
	}
	return models, nil
}

func (s *LaunchService) DryRun(toolName, providerName, model string, extraArgs []string) (LaunchPlanDTO, error) {
	tool, err := loadTool(toolName)
	if err != nil {
		return LaunchPlanDTO{}, err
	}
	file, err := providers.Load(s.providersPath)
	if err != nil {
		return LaunchPlanDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	ep, ok := file.Endpoints[providerName]
	if !ok {
		return LaunchPlanDTO{}, NewError("PROVIDER_NOT_FOUND", "provider not found", map[string]string{"name": providerName})
	}
	launch := tools.ResolveLaunchEnv(tool, ep, providerName, model)
	args := append([]string{}, launch.Inject...)
	args = append(args, extraArgs...)
	return LaunchPlanDTO{
		Tool: toolDTO(tool), Provider: providerDTO(providerName, ep), Model: model,
		Command: tool.LaunchCommand(), Args: args, Environment: launch.Env,
	}, nil
}
