package services

import (
	"fmt"

	"backend/internal/models"
)

type DataForSEOService interface {
	FetchEstimatedTraffic(siteURL string) ([]models.Metric, error)
}

type dataForSEO struct {
	login    string
	password string
}

func NewDataForSEOService(login, password string) DataForSEOService {
	return &dataForSEO{
		login:    login,
		password: password,
	}
}

func (s *dataForSEO) FetchEstimatedTraffic(siteURL string) ([]models.Metric, error) {
	return nil, fmt.Errorf("dataforseo traffic is not connected")
}
