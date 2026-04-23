package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
)

type NLIService struct {
	nliRepo *repository.NLIRepository
}

func NewNLIService() *NLIService {
	return &NLIService{
		nliRepo: repository.NewNLIRepository(),
	}
}

func (s *NLIService) GetChatByID(ctx context.Context, id uuid.UUID) (*models.NLIChat, error) {
	return s.nliRepo.GetByID(ctx, id)
}

func (s *NLIService) ListChatSummariesByHubID(ctx context.Context, hubID uuid.UUID) ([]*models.NLIChatSummary, error) {
	items, err := s.nliRepo.ListSummariesByHubID(ctx, hubID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Title = summarizeTitle(items[i].Title)
	}
	return items, nil
}

func summarizeTitle(firstMessage string) string {
	s := strings.TrimSpace(firstMessage)
	if s == "" {
		return ""
	}

	// If the first message is a JSON string like {"type":"user","data":"..."}, prefer the data field.
	var parsed models.NLIConversationItem
	if err := json.Unmarshal([]byte(s), &parsed); err == nil {
		if t := strings.TrimSpace(parsed.Data); t != "" {
			s = t
		}
	}

	const maxRunes = 80
	r := []rune(s)
	if len(r) > maxRunes {
		return strings.TrimSpace(string(r[:maxRunes])) + "…"
	}
	return s
}
