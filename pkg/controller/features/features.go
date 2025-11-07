package features

import (
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type FeaturesController interface {
	GetUserFeatures(w http.ResponseWriter, r *http.Request)
}

type featuresController struct {
	featuresService service.FeaturesService
	logger          *utils.ErrorLogger
}

func NewFeaturesController() FeaturesController {
	return &featuresController{
		featuresService: service.NewFeaturesService(),
		logger:          utils.NewErrorLogger("features_controller"),
	}
}

// GetFeatures retrieves features based on user auth
// @Summary Get Features by user auth
// @Description Returns the features for the user
// @Tags Features
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "No features found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/features [get]
func (c *featuresController) GetUserFeatures(w http.ResponseWriter, r *http.Request) {
	userFeatures, err := c.featuresService.GetUserFeatures(r.Header.Get(models.HeaderXUser))
	if err != nil {
		c.logger.LogError(err, "Failed to get user features", nil)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    userFeatures,
	})
}
