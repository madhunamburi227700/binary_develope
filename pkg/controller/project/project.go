package project

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// ProjectController handles project HTTP requests
type ProjectController struct {
	projectService *service.ProjectService
	logger         *utils.ErrorLogger
}

// NewProjectController creates a new project controller
func NewProjectController() *ProjectController {
	return &ProjectController{
		projectService: service.NewProjectService(),
		logger:         utils.NewErrorLogger("project_controller"),
	}
}

// // CreateProject creates a new project
// @Summary Create a new project
// @Description Creates a new project with the provided details
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   request body service.CreateProjectRequest true "Project creation details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects [post]
func (c *ProjectController) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req service.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogWarning("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	project, err := c.projectService.CreateProject(r.Context(), &req)
	if err != nil {
		errMsg := "Failed to create project"
		statusCode := http.StatusInternalServerError
		c.logger.LogError(err, errMsg, map[string]interface{}{
			"request": req,
		})

		if strings.Contains(err.Error(), "already exists") {
			errMsg = "Project name already exists for this hub, please try another name."
			statusCode = http.StatusConflict
		}
		utils.SendErrorResponse(w, statusCode, errMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    project,
	})
}

// UpdateProject updates a existing project
// @Summary Update a existing project
// @Description Updates a existing project with the provided details
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   request body service.UpdateProjectRequest true "Project update details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects [put]
func (c *ProjectController) UpdateProject(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogWarning("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.ID = mux.Vars(r)["id"]

	project, err := c.projectService.UpdateProject(r.Context(), &req)
	if err != nil {
		c.logger.LogError(err, "Failed to update project", map[string]interface{}{
			"request": req,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    project,
	})
}

// GetProject retrieves a project by its ID
// @Summary Get project by ID
// @Description Returns the project with the specified ID
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   id path string true "Project ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid project ID"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects/{id} [get]
func (c *ProjectController) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectId, ok := vars["id"]
	if !ok {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	// id, err := uuid.Parse(idStr)
	// if err != nil {
	// 	http.Error(w, "Invalid project ID", http.StatusBadRequest)
	// 	return
	// }

	project, err := c.projectService.GetProject(r.Context(), projectId)
	if err != nil {
		c.logger.LogError(err, "Failed to get project", map[string]interface{}{
			"projectId": projectId,
		})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    project,
	})
}

// GetProjectStats retrieves a project stats by its ID
// @Summary Get project stats by ID
// @Description Returns the project stats with the specified ID
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   id path string true "Project ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid project ID"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects/{id}/stats [get]
func (c *ProjectController) GetProjectStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectId, ok := vars["id"]
	if !ok {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	project, err := c.projectService.GetProjectStats(r.Context(), projectId, r.URL.Query().Get("db"))
	if err != nil {
		c.logger.LogError(err, "Failed to get project stats", map[string]interface{}{
			"projectId": projectId,
		})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    project,
	})
}

// // UpdateProject handles PUT /api/v1/projects/{id}
// func (c *ProjectController) UpdateProject(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	idStr, ok := vars["id"]
// 	if !ok {
// 		http.Error(w, "Project ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	id, err := uuid.Parse(idStr)
// 	if err != nil {
// 		http.Error(w, "Invalid project ID", http.StatusBadRequest)
// 		return
// 	}

// 	var req service.UpdateProjectRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		c.logger.LogWarning("Invalid request body", map[string]interface{}{
// 			"error": err.Error(),
// 		})
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	project, err := c.projectService.UpdateProject(r.Context(), id, &req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to update project", map[string]interface{}{
// 			"project_id": id,
// 			"request":    req,
// 		})
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success": true,
// 		"data":    project,
// 	})
// }

// DeleteProject deletes a project by its ID
// @Summary Delete project by ID
// @Description Deletes the project with the specified ID
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   id path string true "Project ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid project ID"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects/{id} [delete]
func (c *ProjectController) DeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectId, ok := vars["id"]
	if !ok {
		http.Error(w, "Project ID is required", http.StatusBadRequest)
		return
	}

	teamIds := r.URL.Query().Get("teamIds")

	// id, err := uuid.Parse(idStr)
	// if err != nil {
	// 	http.Error(w, "Invalid project ID", http.StatusBadRequest)
	// 	return
	// }

	err := c.projectService.DeleteProject(r.Context(), teamIds, projectId)
	if err != nil {
		c.logger.LogError(err, "Failed to delete project", map[string]interface{}{
			"projectId": projectId,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// // ListProjects handles GET /api/v1/projects
// func (c *ProjectController) ListProjects(w http.ResponseWriter, r *http.Request) {
// 	req := c.buildListRequest(r)

// 	result, err := c.projectService.ListProjects(r.Context(), req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to list projects", map[string]interface{}{
// 			"request": req,
// 		})
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success":    true,
// 		"data":       result.Data,
// 		"pagination": result.Pagination,
// 	})
// }

// // ListProjectsWithDetails handles GET /api/v1/projects/details
// func (c *ProjectController) ListProjectsWithDetails(w http.ResponseWriter, r *http.Request) {
// 	req := c.buildListRequest(r)

// 	result, err := c.projectService.ListProjectsWithDetails(r.Context(), req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to list projects with details", map[string]interface{}{
// 			"request": req,
// 		})
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success":    true,
// 		"data":       result.Data,
// 		"pagination": result.Pagination,
// 	})
// }

// // GetProjectsByHub handles GET /api/v1/hubs/{hub_id}/projects
// func (c *ProjectController) GetProjectsByHub(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	hubIDStr, ok := vars["hub_id"]
// 	if !ok {
// 		http.Error(w, "Hub ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	hubID, err := uuid.Parse(hubIDStr)
// 	if err != nil {
// 		http.Error(w, "Invalid hub ID", http.StatusBadRequest)
// 		return
// 	}

// 	req := c.buildListRequest(r)
// 	result, err := c.projectService.GetProjectsByHub(r.Context(), hubID, req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to get projects by hub", map[string]interface{}{
// 			"hub_id":  hubID,
// 			"request": req,
// 		})
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success":    true,
// 		"data":       result.Data,
// 		"pagination": result.Pagination,
// 	})
// }

// // GetProjectsByIntegration handles GET /api/v1/integrations/{integration_id}/projects
// func (c *ProjectController) GetProjectsByIntegration(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	integrationIDStr, ok := vars["integration_id"]
// 	if !ok {
// 		http.Error(w, "Integration ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	integrationID, err := uuid.Parse(integrationIDStr)
// 	if err != nil {
// 		http.Error(w, "Invalid integration ID", http.StatusBadRequest)
// 		return
// 	}

// 	req := c.buildListRequest(r)
// 	result, err := c.projectService.GetProjectsByIntegration(r.Context(), integrationID, req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to get projects by integration", map[string]interface{}{
// 			"integration_id": integrationID,
// 			"request":        req,
// 		})
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success":    true,
// 		"data":       result.Data,
// 		"pagination": result.Pagination,
// 	})
// }

// buildListRequest builds a list request from HTTP query parameters
func (c *ProjectController) buildListRequest(r *http.Request) *service.GetProjectSummariesForTeamsRequest {
	req := &service.GetProjectSummariesForTeamsRequest{}

	// Parse query parameters
	query := r.URL.Query()

	// Project Name
	req.ProjectName = query.Get("project_name")

	// Pagination
	if pageStr := query.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.PageNo = page
		}
	}

	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			req.PageLimit = pageSize
		}
	}

	return req
}

// GetProjectSummariesForHub returns a list of project summaries for a hub
// @Summary Get project summaries for hub
// @Description Returns a list of project summaries for the specified hub
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   hub_id path string true "Hub ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid hub ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/hubs/{hub_id}/projects/summaries [get]
func (c *ProjectController) GetProjectSummariesForHub(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	vars := mux.Vars(r)
	hubID, ok := vars["hub_id"]
	if !ok {
		http.Error(w, "Hub ID is required", http.StatusBadRequest)
		return
	}

	pageNo := utils.StringToInt(query.Get("pageNo"), 1)
	pageLimit := utils.StringToInt(query.Get("pageLimit"), 10)

	if pageNo < 1 {
		pageNo = 1
	}

	if pageLimit < 1 {
		pageLimit = 10
	}

	result, err := c.projectService.GetProjectSummariesForTeams(r.Context(), hubID, pageNo, pageLimit)
	if err != nil {
		c.logger.LogError(err, "Failed to get project summaries for hub", map[string]interface{}{
			"hubID":     hubID,
			"pageNo":    pageNo,
			"pageLimit": pageLimit,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// func (c *ProjectController) GetProjectDetails(w http.ResponseWriter, r *http.Request) {
// 	req := &client.ProjectDetailsRequest{}
// 	vars := mux.Vars(r)

// 	query := r.URL.Query()
// 	hubIDstr := query.Get("hubId")

// 	if hubIDstr == "" {
// 		http.Error(w, "Hub ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	hubIDs := strings.Split(hubIDstr, ",")
// 	for _, hubID := range hubIDs {
// 		_, err := uuid.Parse(hubID)
// 		if err != nil {
// 			http.Error(w, "Invalid hub ID", http.StatusBadRequest)
// 			return
// 		}
// 	}

// 	req.TeamIDs = hubIDstr

// 	projectIDStr, ok := vars["project_id"]
// 	if !ok {
// 		http.Error(w, "Project ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	req.ProjectID = projectIDStr
// 	result, err := c.projectService.GetProjectDetails(r.Context(), req)
// 	if err != nil {
// 		c.logger.LogError(err, "Failed to get project details", map[string]interface{}{
// 			"request": req,
// 		})
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"success": true,
// 		"data":    result,
// 	})
// }

// GetProjectSummaryCount returns the count of project summaries
// @Summary Get project summary count
// @Description Returns the count of project summaries based on filters
// @Tags Projects
// @Accept  json
// @Produce  json
// @Param   team_id query string false "Filter by team ID"
// @Param   hub_id query string false "Filter by hub ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/projects/summaries/count [get]
func (c *ProjectController) GetProjectSummaryCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hubIDStr, ok := vars["hub_id"]
	if !ok {
		http.Error(w, "Hub ID is required", http.StatusBadRequest)
		return
	}

	// hub ids will be comma separated
	hubIDs := strings.Split(hubIDStr, ",")
	for _, hubID := range hubIDs {
		_, err := uuid.Parse(hubID)
		if err != nil {
			http.Error(w, "Invalid hub ID", http.StatusBadRequest)
			return
		}
	}

	result, err := c.projectService.GetProjectSummaryCount(r.Context(), hubIDs)
	if err != nil {
		c.logger.LogError(err, "Failed to get project summary count", map[string]interface{}{
			"request": hubIDs,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}
