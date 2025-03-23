package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

const template = "Погода в городе: %s\n%.1f градусов\n%s"

type MainResponse struct {
	Temp float64 `json:"temp"`
}

type WResponse struct {
	Desc string `json:"description"`
}

type WeatherResponse struct {
	Main    MainResponse `json:"main"`
	City    string       `json:"name"`
	Weather []WResponse  `json:"weather"`
}

func buildAnswer(resp *WeatherResponse) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("got empty response")
	}

	if len(resp.Weather) == 0 {
		return "", fmt.Errorf("got empty weather in response")
	}

	return fmt.Sprintf(template, resp.City, resp.Main.Temp, resp.Weather[0].Desc), nil
}

func buildQuery() url.URL {
	api_key := os.Getenv("WEATHER_API_KEY")
	dst := url.URL{
		Scheme: "https",
		Host:   "api.openweathermap.org",
		Path:   "data/2.5/weather",
	}
	dst_query := dst.Query()
	dst_query.Set("q", "moscow")
	dst_query.Set("lang", "ru")
	dst_query.Set("units", "metric")
	dst_query.Set("appid", api_key)

	dst.RawQuery = dst_query.Encode()
	return dst
}

func WeatherHandle(context tele.Context) error {

	query := buildQuery()

	log.Infof("Get query: %s", query.String())

	r, err := http.Get(query.String())

	if err != nil {
		return err
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)

	if err != nil {
		return err
	}

	var resp WeatherResponse

	json.Unmarshal(body, &resp)

	log.Infof("Status: %s Data: %s", r.Status, body)
	log.Infof("Parsed: %v", resp)

	answer, err := buildAnswer(&resp)

	if err != nil {
		return err
	}
	context.Send(answer)

	return nil
}
