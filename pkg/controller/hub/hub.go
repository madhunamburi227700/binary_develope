package hub

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// HubController handles hub HTTP requests
type HubController struct {
	hubService *service.HubService
	logger     *utils.ErrorLogger
}

// NewHubController creates a new hub controller
func NewHubController() *HubController {
	return &HubController{
		hubService: service.NewHubService(),
		logger:     utils.NewErrorLogger("hub_controller"),
	}
}

// buildListRequest builds a list request from HTTP query parameters
// func (c *HubController) buildListRequest(r *http.Request) (*service.HubListRequest, error) {
// 	req := &service.HubListRequest{}
// 	query := r.URL.Query()
// 	vars := mux.Vars(r)

// 	// Owner ID - check path parameter first, then query parameter
// 	var ownerIDStr string
// 	if ownerIDStr = vars["email"]; ownerIDStr == "" {
// 		ownerIDStr = query.Get("email")
// 	}

// 	if ownerIDStr == "" {
// 		return nil, fmt.Errorf("owner_id is required")
// 	}

// 	ownerID, err := uuid.Parse(ownerIDStr)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid owner_id format: %v", err)
// 	}
// 	req.OwnerID = &ownerID

// 	// // Collaborator ID
// 	// if collaboratorIDStr := query.Get("collaborator_id"); collaboratorIDStr != "" {
// 	// 	if collaboratorID, err := uuid.Parse(collaboratorIDStr); err == nil {
// 	// 		req.CollaboratorID = &collaboratorID
// 	// 	}
// 	// }

// 	// Search
// 	req.Search = query.Get("search")

// 	// Pagination
// 	if pageStr := query.Get("page"); pageStr != "" {
// 		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
// 			req.Page = page
// 		}
// 	}
// 	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
// 		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
// 			req.PageSize = pageSize
// 		}
// 	}

// 	// Ordering
// 	req.OrderBy = query.Get("order_by")
// 	req.OrderDir = query.Get("order_dir")

// 	return req, nil
// }

// CreateHub creates a new hub
// @Summary Create a new hub
// @Description Creates a new hub with the provided details
// @Tags Hubs
// @Accept  json
// @Produce  json
// @Param   request body service.CreateHubRequest true "Hub creation details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/hubs [post]
func (c *HubController) CreateHub(w http.ResponseWriter, r *http.Request) {

	var req service.CreateHubRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.logger.LogWarning("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hub, err := c.hubService.CreateHub(r.Context(), &req)
	if err != nil {
		c.logger.LogError(err, "Failed to create hub", map[string]interface{}{
			"request": req,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    hub,
	})
}

type ListHubsByOwnerRequest struct {
	Email string `json:"email"`
}

// ListHubsByOwner returns a list of hubs owned by the specified user
// @Summary List hubs by owner
// @Description Returns a list of hubs owned by the specified user
// @Tags Hubs
// @Accept  json
// @Produce  json
// @Param   email query string true "Email of the hub owner"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Missing or invalid email parameter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/hubs/user/list [get]
func (c *HubController) ListHubsByOwner(w http.ResponseWriter, r *http.Request) {
	// Get email filter from query parameter
	email := r.URL.Query().Get("email")
	if email == "" {
		c.logger.LogWarning("Email is required", map[string]interface{}{
			"request": email,
		})
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	result, err := c.hubService.List(r.Context(), email)
	if err != nil {
		c.logger.LogError(err, "Failed to list hubs", map[string]interface{}{
			"request": email,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result.Hubs,
	})
}

// GetHub retrieves a hub by its ID
// @Summary Get hub by ID
// @Description Returns the hub with the specified ID
// @Tags Hubs
// @Accept  json
// @Produce  json
// @Param   id path string true "Hub ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid hub ID"
// @Failure 404 {object} map[string]string "Hub not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/hubs/{id} [get]
func (c *HubController) GetHub(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hubIDStr, ok := vars["id"]
	if !ok {
		http.Error(w, "Hub ID is required", http.StatusBadRequest)
		return
	}

	hubID, err := uuid.Parse(hubIDStr)
	if err != nil {
		http.Error(w, "Invalid hub ID", http.StatusBadRequest)
		return
	}

	hub, err := c.hubService.GetByID(r.Context(), hubID.String())
	if err != nil {
		c.logger.LogError(err, "Failed to get hub", map[string]interface{}{
			"hub_id": hubID,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    hub,
	})
}
