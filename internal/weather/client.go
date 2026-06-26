package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	APIKey string
}

type CurrentWeather struct {
	Location    string
	Temperature float64
	Condition   string
	WindKPH     float64
	Humidity    int
}

func (c *Client) Current(ctx context.Context, location string) (string, error) {
	u := fmt.Sprintf(
		"https://api.weatherapi.com/v1/current.json?key=%s&q=%s",
		url.QueryEscape(c.APIKey),
		url.QueryEscape(location),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Location struct {
			Name string `json:"name"`
		} `json:"location"`

		Current struct {
			TempC    float64 `json:"temp_c"`
			WindKPH  float64 `json:"wind_kph"`
			Humidity int     `json:"humidity"`

			Condition struct {
				Text string `json:"text"`
			} `json:"condition"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Location: %s\nTemperature: %.1f°C\nCondition: %s\nWind: %.1f kph\nHumidity: %d%%",
		data.Location.Name,
		data.Current.TempC,
		data.Current.Condition.Text,
		data.Current.WindKPH,
		data.Current.Humidity,
	), nil
}

func (c *Client) Forecast(ctx context.Context, location string, days int) (string, error) {
	u := fmt.Sprintf(
		"https://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=%d",
		url.QueryEscape(c.APIKey),
		url.QueryEscape(location),
		days,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Location struct {
			Name string `json:"name"`
		} `json:"location"`

		Forecast struct {
			ForecastDay []struct {
				Date string `json:"date"`

				Day struct {
					AvgTempC float64 `json:"avgtemp_c"`

					Condition struct {
						Text string `json:"text"`
					} `json:"condition"`
				} `json:"day"`
			} `json:"forecastday"`
		} `json:"forecast"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	var lines []string

	for _, day := range data.Forecast.ForecastDay {
		lines = append(lines,
			fmt.Sprintf("• %s: %s, %.0f°C",
				day.Date,
				strings.ToLower(day.Day.Condition.Text),
				day.Day.AvgTempC,
			),
		)
	}

	return fmt.Sprintf(
		"%s %d-day forecast:\n%s",
		data.Location.Name,
		len(data.Forecast.ForecastDay),
		strings.Join(lines, "\n"),
	), nil
}
