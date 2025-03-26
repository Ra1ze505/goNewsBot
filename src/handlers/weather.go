package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Ra1ze505/goNewsBot/src/keyboard"
	"github.com/Ra1ze505/goNewsBot/src/repository"

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

func buildQuery(city string) url.URL {
	api_key := os.Getenv("WEATHER_API_KEY")
	dst := url.URL{
		Scheme: "https",
		Host:   "api.openweathermap.org",
		Path:   "data/2.5/weather",
	}
	dst_query := dst.Query()
	dst_query.Set("q", city)
	dst_query.Set("lang", "ru")
	dst_query.Set("units", "metric")
	dst_query.Set("appid", api_key)

	dst.RawQuery = dst_query.Encode()
	return dst
}

func WeatherHandle(context tele.Context) error {
	user, ok := context.Get("user").(*repository.User)
	if !ok {
		context.Send("Что-то пошло не так, попробуйте позже", keyboard.GetStartKeyboard())
		return fmt.Errorf("user not found in context")
	}

	query := buildQuery(user.City)

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
	answer, err := buildAnswer(&resp)

	if err != nil {
		return err
	}

	return context.Send(answer, keyboard.GetStartKeyboard())
}
