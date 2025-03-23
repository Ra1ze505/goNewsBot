package handlers

import (
	"net/http"
	"net/url"
	"os"
	"io"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"
)

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

	resp, err := http.Get(query.String())

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	data := json.NewDecoder(resp.Body)
	data.Decode()

	log.Infof("Status: %s Data: %s", resp.Status, body)

	return context.Send("pu")

}
