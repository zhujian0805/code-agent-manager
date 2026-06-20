package sidecar

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
	"github.com/chat2anyllm/code-agent-manager/internal/instructions"
)

// instructionsStore constructs a store over the default cam.db. Mirrors the
// metadata handlers, which build their store per request.
func instructionsStore() *instructions.Store { return instructions.New("") }

// handleInstructions serves the collection endpoints: list and create.
func (s *Server) handleInstructions(w http.ResponseWriter, r *http.Request) {
	store := instructionsStore()
	switch r.Method {
	case http.MethodGet:
		list, err := store.ListWithInstalls(r.Context())
		if err != nil {
			writeInstructionError(w, err)
			return
		}
		if list == nil {
			list = []instructions.Instruction{}
		}
		writeJSON(w, list)
	case http.MethodPost:
		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := store.Create(r.Context(), input.Name, input.Description, input.Content)
		if err != nil {
			writeInstructionError(w, err)
			return
		}
		writeJSONStatus(w, http.StatusCreated, created)
	default:
		methodNotAllowed(w)
	}
}

// handleInstructionsSub routes the sub-paths under /api/instructions/.
func (s *Server) handleInstructionsSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/instructions/"), "/")
	parts := strings.Split(rest, "/")

	switch {
	case rest == "targets":
		s.handleInstructionTargets(w, r)
	case len(parts) == 2 && parts[0] == "installs":
		s.handleInstructionInstallDelete(w, r, parts[1])
	case len(parts) == 2 && parts[1] == "installs":
		s.handleInstructionInstallCreate(w, r, parts[0])
	case len(parts) == 1 && parts[0] != "":
		s.handleInstructionItem(w, r, parts[0])
	default:
		writeError(w, http.StatusNotFound, "instruction not found")
	}
}

func (s *Server) handleInstructionItem(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "instruction not found")
		return
	}
	store := instructionsStore()
	switch r.Method {
	case http.MethodGet:
		in, err := store.Get(r.Context(), id)
		if err != nil {
			writeInstructionError(w, err)
			return
		}
		writeJSON(w, in)
	case http.MethodPut:
		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := store.Update(r.Context(), id, input.Name, input.Description, input.Content)
		if err != nil {
			writeInstructionError(w, err)
			return
		}
		writeJSON(w, updated)
	case http.MethodDelete:
		if err := store.Delete(r.Context(), id); err != nil {
			writeInstructionError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleInstructionInstallCreate(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "instruction not found")
		return
	}
	var input struct {
		App        string `json:"app"`
		Level      string `json:"level"`
		ProjectDir string `json:"project_dir"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.Level == "" {
		input.Level = string(entities.InstallLevelUser)
	}
	store := instructionsStore()
	install, err := store.Install(r.Context(), id, input.App, input.Level, input.ProjectDir)
	if err != nil {
		writeInstructionError(w, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, install)
}

func (s *Server) handleInstructionInstallDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "install not found")
		return
	}
	store := instructionsStore()
	if err := store.Uninstall(r.Context(), id); err != nil {
		writeInstructionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// InstructionTarget describes an app and the install levels it supports.
type InstructionTarget struct {
	App      string          `json:"app"`
	Supports map[string]bool `json:"supports"`
}

func (s *Server) handleInstructionTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	apps := entities.InstructionApps()
	sort.Strings(apps)
	out := make([]InstructionTarget, 0, len(apps))
	for _, app := range apps {
		supports := map[string]bool{"user": false, "project": false}
		for _, lvl := range entities.InstructionAppLevels(app) {
			supports[string(lvl)] = true
		}
		out = append(out, InstructionTarget{App: app, Supports: supports})
	}
	writeJSON(w, out)
}

// writeInstructionError maps instructions package errors to HTTP status codes.
func writeInstructionError(w http.ResponseWriter, err error) {
	var conflict *instructions.ConflictError
	switch {
	case errors.As(err, &conflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, instructions.ErrDuplicateName):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, instructions.ErrInstructionNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, instructions.ErrInvalidName),
		errors.Is(err, instructions.ErrProjectDirRequired),
		errors.Is(err, instructions.ErrUnsupportedTarget):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}
