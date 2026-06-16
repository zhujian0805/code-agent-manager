package desktop

// Services groups all desktop API services for Wails registration or tests.
type Services struct {
	App       *AppService
	Providers *ProviderService
	MCP       *MCPService
	Entities  *EntityService
	Tools     *ToolService
	Doctor    *DoctorService
	Config    *ConfigService
	Launch    *LaunchService
}

func NewServices(version, providersPath string) Services {
	return Services{
		App:       NewAppService(version),
		Providers: NewProviderService(providersPath),
		MCP:       NewMCPService(),
		Entities:  NewEntityService(),
		Tools:     NewToolService(),
		Doctor:    NewDoctorService(version, providersPath),
		Config:    NewConfigService(),
		Launch:    NewLaunchService(providersPath),
	}
}
