package repository

//go:generate mockgen -source=weather.go -destination=../mocks/repository/weather_mock.go

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type WeatherRepositoryInterface interface {
	GetWeatherByCity(city string) (*WeatherResponse, error)
}

type WeatherRepository struct {
	apiKey string
}

func NewWeatherRepository() WeatherRepositoryInterface {
	return &WeatherRepository{
		apiKey: os.Getenv("WEATHER_API_KEY"),
	}
}

type MainResponse struct {
	Temp float64 `json:"temp"`
}

type WResponse struct {
	Desc string `json:"description"`
}

type WeatherResponse struct {
	Main     MainResponse `json:"main"`
	City     string       `json:"name"`
	Weather  []WResponse  `json:"weather"`
	Timezone int          `json:"timezone"`
}

func (r *WeatherRepository) GetWeatherByCity(city string) (*WeatherResponse, error) {
	query := r.buildQuery(city)

	resp, err := http.Get(query.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("city not found or API error: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var weatherResp WeatherResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &weatherResp, nil
}

func (r *WeatherRepository) buildQuery(city string) url.URL {
	dst := url.URL{
		Scheme: "https",
		Host:   "api.openweathermap.org",
		Path:   "data/2.5/weather",
	}
	dst_query := dst.Query()
	dst_query.Set("q", city)
	dst_query.Set("lang", "ru")
	dst_query.Set("units", "metric")
	dst_query.Set("appid", r.apiKey)

	dst.RawQuery = dst_query.Encode()
	return dst
}
