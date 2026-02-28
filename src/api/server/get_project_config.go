package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/config"
)

// GetProjectConfig returns the parsed project.critic configuration.
func (s *Server) GetProjectConfig(
	ctx context.Context,
	req *connect.Request[api.GetProjectConfigRequest],
) (*connect.Response[api.GetProjectConfigResponse], error) {
	response := depanic(func() (*api.GetProjectConfigResponse, error) {
		return getProjectConfigImpl(s)
	})
	return connect.NewResponse(response), nil
}

func getProjectConfigImpl(server *Server) (*api.GetProjectConfigResponse, error) {
	pc := server.config.ProjectConfig
	if pc == nil {
		pc = config.DefaultProjectConfig()
	}

	categories := make([]*api.FileCategory, 0, len(pc.Categories))
	for _, cat := range pc.Categories {
		categories = append(categories, &api.FileCategory{
			Name:     cat.Name,
			Patterns: cat.Patterns,
		})
	}

	var editor *api.EditorConfig
	if pc.Editor.URL != "" {
		editor = &api.EditorConfig{
			Url: pc.Editor.URL,
		}
	}

	return &api.GetProjectConfigResponse{
		ProjectName: pc.Project.Name,
		Categories:  categories,
		Editor:      editor,
	}, nil
}
