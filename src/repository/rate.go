package repository

//go:generate mockgen -source=rate.go -destination=../mocks/repository/rate_mock.go -package=mock_repository

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Rate struct {
	Date time.Time
	Data json.RawMessage
}

type CurrencyRate struct {
	ID       string  `json:"ID"`
	NumCode  string  `json:"NumCode"`
	CharCode string  `json:"CharCode"`
	Nominal  int     `json:"Nominal"`
	Name     string  `json:"Name"`
	Value    float64 `json:"Value"`
	Previous float64 `json:"Previous"`
}

type Rates struct {
	USD CurrencyRate `json:"USD"`
	EUR CurrencyRate `json:"EUR"`
	// Add more currencies as needed
}

type RateRepositoryInterface interface {
	SaveRate(rate *Rate) error
	GetLatestRate() (*Rate, error)
	GetRates() (*Rates, error)
}

type RateRepository struct {
	db *sql.DB
}

func NewRateRepository(db *sql.DB) *RateRepository {
	return &RateRepository{db: db}
}

func (r *RateRepository) SaveRate(rate *Rate) error {
	query := `
		INSERT INTO rates (date, data)
		VALUES ($1, $2)
		ON CONFLICT (date) DO NOTHING`

	_, err := r.db.Exec(query, rate.Date, rate.Data)
	return err
}

func (r *RateRepository) GetLatestRate() (*Rate, error) {
	query := `
		SELECT date, data
		FROM rates
		ORDER BY date DESC
		LIMIT 1`

	var rate Rate
	err := r.db.QueryRow(query).Scan(&rate.Date, &rate.Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *RateRepository) GetRates() (*Rates, error) {
	rate, err := r.GetLatestRate()
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, sql.ErrNoRows
	}

	var rates Rates
	if err := json.Unmarshal(rate.Data, &rates); err != nil {
		return nil, err
	}

	return &rates, nil
}
