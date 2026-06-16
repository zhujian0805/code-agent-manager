package desktop

import (
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
)

type EntityService struct{}

func NewEntityService() *EntityService { return &EntityService{} }

func (s *EntityService) List(kind string) ([]EntityDTO, error) {
	store, err := entityStore(kind)
	if err != nil {
		return nil, err
	}
	all, err := store.All()
	if err != nil {
		return nil, wrapError("ENTITY_LIST_FAILED", err)
	}
	out := make([]EntityDTO, 0, len(all))
	for _, entity := range all {
		out = append(out, entityDTO(entity))
	}
	return out, nil
}

func (s *EntityService) Search(kind, query string) ([]EntityDTO, error) {
	all, err := s.List(kind)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return all, nil
	}
	out := []EntityDTO{}
	for _, entity := range all {
		if strings.Contains(strings.ToLower(entity.Name), q) ||
			strings.Contains(strings.ToLower(entity.Description), q) ||
			strings.Contains(strings.ToLower(strings.Join(entity.Tags, " ")), q) {
			out = append(out, entity)
		}
	}
	return out, nil
}

func (s *EntityService) Save(kind string, input EntityDTO) (EntityDTO, error) {
	store, err := entityStore(kind)
	if err != nil {
		return EntityDTO{}, err
	}
	entity := entities.Entity{
		Name: input.Name, Description: input.Description, Content: input.Content, Path: input.Path,
		Apps: input.Apps, Tags: input.Tags, Metadata: input.Metadata,
	}
	if err := store.Put(entity); err != nil {
		return EntityDTO{}, wrapError("ENTITY_SAVE_FAILED", err)
	}
	saved, err := store.Get(input.Name)
	if err != nil {
		return EntityDTO{}, wrapError("ENTITY_LOAD_FAILED", err)
	}
	return entityDTO(saved), nil
}

func (s *EntityService) Uninstall(kind, name string) (OperationResult, error) {
	store, err := entityStore(kind)
	if err != nil {
		return OperationResult{}, err
	}
	removed, err := store.Delete(name)
	if err != nil {
		return OperationResult{}, wrapError("ENTITY_DELETE_FAILED", err)
	}
	if !removed {
		return OperationResult{}, NewError("ENTITY_NOT_FOUND", "entity not found", map[string]string{"kind": kind, "name": name})
	}
	return OperationResult{OK: true, Message: "entity removed", Path: store.Path()}, nil
}

func (s *EntityService) Install(kind string, input EntityDTO) (EntityDTO, error) {
	return s.Save(kind, input)
}

func (s *EntityService) Update(kind string) (OperationResult, error) {
	if _, err := parseKind(kind); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{OK: true, Message: "repository update is available from the CLI and will be wired to desktop fetch flows incrementally"}, nil
}

func entityStore(kind string) (*entities.Store, error) {
	parsed, err := parseKind(kind)
	if err != nil {
		return nil, err
	}
	return entities.NewStore(parsed), nil
}

func parseKind(kind string) (entities.Kind, error) {
	switch entities.Kind(kind) {
	case entities.KindPrompt, entities.KindSkill, entities.KindAgent, entities.KindPlugin:
		return entities.Kind(kind), nil
	default:
		return "", NewError("ENTITY_KIND_INVALID", "unsupported entity kind", map[string]string{"kind": kind})
	}
}

func entityDTO(entity entities.Entity) EntityDTO {
	return EntityDTO{
		Kind: string(entity.Kind), Name: entity.Name, Description: entity.Description,
		Content: entity.Content, Path: entity.Path, Apps: entity.Apps, Tags: entity.Tags,
		Metadata: entity.Metadata, UpdatedAt: entity.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
