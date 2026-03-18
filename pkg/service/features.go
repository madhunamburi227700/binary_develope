package service

import (
	"context"
	"slices"

	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type FeaturesService interface {
	GetUserFeatures(username string) (map[string]string, error)
}

type featuresService struct {
	logger   *utils.ErrorLogger
	userRepo *repository.UserRepository
}

func NewFeaturesService() FeaturesService {
	return &featuresService{
		logger:   utils.NewErrorLogger("features_service"),
		userRepo: repository.NewUserRepository(),
	}
}

func (f *featuresService) GetUserFeatures(username string) (map[string]string, error) {
	userFeatures := map[string]string{}
	user, err := f.userRepo.GetByProviderUserID(context.TODO(), username)
	if err != nil {
		return userFeatures, err
	}

	// audit features flag addition
	if slices.Contains(config.GetAuditUsers(), user.Email.String) {
		userFeatures["audit"] = "true"
	}

	// chatInterface features flag addition true for all users for now
	userFeatures["chatInterface"] = "true"

	if slices.Contains(config.GetCSPMUsers(), user.Email.String) {
		userFeatures["cspm"] = "true"
	}

	return userFeatures, nil
}
