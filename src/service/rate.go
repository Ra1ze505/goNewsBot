package service

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Ra1ze505/goNewsBot/src/repository"
	log "github.com/sirupsen/logrus"
)

type CBRResponse struct {
	Timestamp string          `json:"Timestamp"`
	Valute    json.RawMessage `json:"Valute"`
}

type RateService struct {
	repo *repository.RateRepository
}

func NewRateService(repo *repository.RateRepository) *RateService {
	return &RateService{repo: repo}
}

func (s *RateService) FetchAndSaveRates() error {
	resp, err := http.Get("https://www.cbr-xml-daily.ru/daily_json.js")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cbrResp CBRResponse
	if err := json.Unmarshal(body, &cbrResp); err != nil {
		return err
	}

	timestamp, err := time.Parse(time.RFC3339, cbrResp.Timestamp)
	if err != nil {
		return err
	}

	rate := &repository.Rate{
		Date: timestamp,
		Data: cbrResp.Valute,
	}

	return s.repo.SaveRate(rate)
}

func (s *RateService) StartRateFetcher() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			if err := s.FetchAndSaveRates(); err != nil {
				log.Errorf("Error fetching and saving rates: %v", err)
			} else {
				log.Info("Rates fetched and saved successfully")
			}
			<-ticker.C
		}
	}()
}
