package nli

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type NLIController struct {
	nliService *service.NLIService
	logger     *utils.ErrorLogger
}

func NewNLIController() *NLIController {
	return &NLIController{
		nliService: service.NewNLIService(),
		logger:     utils.NewErrorLogger("nli_controller"),
	}
}

// ListChatsByHubID returns all nli chats for a hub.
// @Summary List NLI chats for a hub
// @Description Returns all chats from nli table filtered by hub_id.
// @Tags NLI
// @Accept */*
// @Produce json
// @Param hub_id path string true "Hub ID (uuid)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid hub_id"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/nli/chats/hub/{hub_id} [get]
func (c *NLIController) ListChatsByHubID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hubIDStr := vars["hub_id"]
	if hubIDStr == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "hub_id is required")
		return
	}

	hubID, err := uuid.Parse(hubIDStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid hub_id")
		return
	}

	items, err := c.nliService.ListChatSummariesByHubID(r.Context(), hubID)
	if err != nil {
		c.logger.LogError(err, "Failed to list nli chats", map[string]interface{}{"hub_id": hubIDStr})
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to list nli chats")
		return
	}

	utils.SendSuccessResponse(w, items, "NLI chats fetched successfully")
}

// GetChatByID returns one nli chat detail by id.
// @Summary Get NLI chat detail
// @Description Returns chat from nli table for the given id.
// @Tags NLI
// @Accept */*
// @Produce json
// @Param id path string true "Chat ID (uuid)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid id"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/nli/chats/{id} [get]
func (c *NLIController) GetChatByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid id")
		return
	}

	chat, err := c.nliService.GetChatByID(r.Context(), id)
	if err != nil {
		// BaseRepository returns "record not found" string for pgx.ErrNoRows
		if err.Error() == "record not found" {
			utils.SendErrorResponse(w, http.StatusNotFound, "chat not found")
			return
		}
		c.logger.LogError(err, "Failed to get nli chat by id", map[string]interface{}{"id": idStr})
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to get chat")
		return
	}

	utils.SendSuccessResponse(w, chat, "NLI chat fetched successfully")
}
