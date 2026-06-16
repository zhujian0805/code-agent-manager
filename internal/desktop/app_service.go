package desktop

import "runtime"

type AppService struct {
	version string
}

func NewAppService(version string) *AppService { return &AppService{version: version} }

func (s *AppService) Version() string {
	if s.version == "" {
		return "dev"
	}
	return s.version
}

func (s *AppService) Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func (s *AppService) Quit() OperationResult {
	return OperationResult{OK: true, Message: "quit requested"}
}
