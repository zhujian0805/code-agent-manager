package desktop

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chat2anyllm/code-agent-manager/internal/providers"
)

type ProviderService struct {
	path string
	mu   sync.Mutex
}

func NewProviderService(path string) *ProviderService {
	return &ProviderService{path: path}
}

func (s *ProviderService) providersPath() string {
	if s.path != "" {
		return s.path
	}
	return providers.DefaultPath()
}

func (s *ProviderService) Init() (OperationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, created, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return OperationResult{}, wrapError("PROVIDER_INIT_FAILED", err)
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return OperationResult{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	msg := "providers file ready"
	if created {
		msg = "providers file created"
	}
	return OperationResult{OK: true, Message: msg, Path: s.providersPath()}, nil
}

func (s *ProviderService) List() ([]ProviderDTO, error) {
	file, err := providers.Load(s.providersPath())
	if err != nil {
		return nil, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	out := make([]ProviderDTO, 0, len(file.Endpoints))
	for _, name := range file.SortedNames() {
		out = append(out, providerDTO(name, file.Endpoints[name]))
	}
	return out, nil
}

func (s *ProviderService) Show(name string) (ProviderDTO, error) {
	file, err := providers.Load(s.providersPath())
	if err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	ep, ok := file.Endpoints[name]
	if !ok {
		return ProviderDTO{}, NewError("PROVIDER_NOT_FOUND", "provider not found", map[string]string{"name": name})
	}
	return providerDTO(name, ep), nil
}

func (s *ProviderService) Add(input ProviderInput) (ProviderDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, _, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	ep := endpointFromInput(input)
	if err := providers.Add(&file, input.Name, ep); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_ADD_FAILED", err)
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	return providerDTO(input.Name, ep), nil
}

func (s *ProviderService) Update(name string, input ProviderInput) (ProviderDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, _, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	patch := providers.Patch{
		Endpoint:        strPtr(input.Endpoint),
		APIKeyEnv:       strPtr(input.APIKeyEnv),
		Description:     strPtr(input.Description),
		ListModelsCmd:   strPtr(input.ListModelsCmd),
		KeepProxyConfig: boolPtr(input.KeepProxyConfig),
		UseProxy:        boolPtr(input.UseProxy),
		Enabled:         input.Enabled,
		Clients:         &providers.ListPatch{Op: providers.ListOpReplace, Items: clientsFromInput(input)},
		Models:          &providers.ListPatch{Op: providers.ListOpReplace, Items: input.Models},
	}
	if err := providers.Update(&file, name, patch); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_UPDATE_FAILED", err)
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	return providerDTO(name, file.Endpoints[name]), nil
}

func (s *ProviderService) Remove(name string) (OperationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, _, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return OperationResult{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	if !providers.Remove(&file, name) {
		return OperationResult{}, NewError("PROVIDER_NOT_FOUND", "provider not found", map[string]string{"name": name})
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return OperationResult{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	return OperationResult{OK: true, Message: "provider removed", Path: s.providersPath()}, nil
}

func (s *ProviderService) Enable(name string) (ProviderDTO, error) {
	return s.setEnabled(name, true)
}

func (s *ProviderService) Disable(name string) (ProviderDTO, error) {
	return s.setEnabled(name, false)
}

func (s *ProviderService) Rename(oldName, newName string) (ProviderDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, _, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	if err := providers.Rename(&file, oldName, newName); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_RENAME_FAILED", err)
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	return providerDTO(newName, file.Endpoints[newName]), nil
}

func (s *ProviderService) ResolveModels(name string) ([]string, error) {
	file, err := providers.Load(s.providersPath())
	if err != nil {
		return nil, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	ep, ok := file.Endpoints[name]
	if !ok {
		return nil, NewError("PROVIDER_NOT_FOUND", "provider not found", map[string]string{"name": name})
	}
	models, err := providers.ResolveModels(ep, name, 24*time.Hour, "", os.Getenv)
	if err != nil {
		return nil, wrapError("MODEL_DISCOVERY_FAILED", err)
	}
	return models, nil
}

func (s *ProviderService) setEnabled(name string, enabled bool) (ProviderDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, _, err := providers.LoadOrInit(s.providersPath())
	if err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_LOAD_FAILED", err)
	}
	if err := providers.SetEnabled(&file, name, enabled); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_UPDATE_FAILED", err)
	}
	if err := providers.Save(s.providersPath(), file); err != nil {
		return ProviderDTO{}, wrapError("PROVIDER_SAVE_FAILED", err)
	}
	return providerDTO(name, file.Endpoints[name]), nil
}

func providerDTO(name string, ep providers.Endpoint) ProviderDTO {
	apiKey := providers.ResolveAPIKey(ep, os.Getenv)
	return ProviderDTO{
		Name:            name,
		Endpoint:        ep.Endpoint,
		APIKeyEnv:       ep.APIKeyEnv,
		SupportedClient: ep.SupportedClient,
		Clients:         ep.Clients(),
		Models:          append([]string(nil), ep.Models...),
		KeepProxyConfig: ep.KeepProxyConfig,
		UseProxy:        ep.UseProxy,
		Enabled:         ep.IsEnabled(),
		Description:     ep.Description,
		MaskedAPIKey:    providers.MaskedAPIKey(apiKey),
	}
}

func endpointFromInput(input ProviderInput) providers.Endpoint {
	clients := clientsFromInput(input)
	return providers.Endpoint{
		Endpoint:        input.Endpoint,
		APIKeyEnv:       input.APIKeyEnv,
		SupportedClient: strings.Join(clients, ","),
		ListModelsCmd:   input.ListModelsCmd,
		Models:          append([]string(nil), input.Models...),
		KeepProxyConfig: input.KeepProxyConfig,
		UseProxy:        input.UseProxy,
		Enabled:         input.Enabled,
		Description:     input.Description,
	}
}

func clientsFromInput(input ProviderInput) []string {
	if len(input.Clients) > 0 {
		return input.Clients
	}
	if input.SupportedClient == "" {
		return nil
	}
	parts := strings.Split(input.SupportedClient, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }
