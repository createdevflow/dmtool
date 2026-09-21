package services

import (
	"fmt"

	"backend/internal/models"
)

type RapidAPIService interface {
	FetchInstagramProfile(username string) (*models.SocialMetric, error)
}

type rapidAPI struct {
	apiKey string
}

func NewRapidAPIService(apiKey string) RapidAPIService {
	return &rapidAPI{apiKey: apiKey}
}

func (s *rapidAPI) FetchInstagramProfile(username string) (*models.SocialMetric, error) {
	return nil, fmt.Errorf("rapidapi instagram is not connected")
}
