package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"weather/internal/domain"
)

type Request struct {
	client  *http.Client
	baseURL string
}

func New() *Request {
	url := "https://api.openweathermap.org/data/2.5/forecast"
	return &Request{
		client:  &http.Client{},
		baseURL: url,
	}
}

var (
	ErrNoApiToken    = errors.New("Api Token не добавлен в систему")
	ErrNoActiveToken = errors.New("Unable to connect to service")
)

// GetApiToken - проверяет, есть ли api token в env файле
// Возвращает Err
func (r *Request) getApiToken() (string, error) {
	token := os.Getenv("API_TOKEN")

	if token == "" {
		return "", ErrNoApiToken
	}

	return token, nil
}

// Возвращает отсортированный слайс с по датам
func (r *Request) GetCityInfo(name string, count int) ([]domain.Weather, error) {
	token, err := r.getApiToken()
	if err != nil {
		return nil, err
	}

	// Допустим пользователь хочет получить данные на 3 дня
	// Тогда нужно сделать так, чтобы точно было на один день
	// больше данных

	url := fmt.Sprintf("%s?q=%s&units=metric&cnt=%d&lang=%s&appid=%s",
		r.baseURL,
		name,
		count*8,
		"ru",
		token,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		List []struct {
			Date        string `json:"dt_txt"`
			Coordinates struct {
				Lon float32 `json:"lon"`
				Lay float32 `json:"lat"`
			} `json:"coord"`
			Weather []struct {
				Main        string `json:"main"`        // clouds
				Description string `json:"description"` // облачно с прояснениями
			} `json:"weather"`
			Temperature struct {
				Temp      float32 `json:"temp"`
				FeelsLike float32 `json:"feels_like"`
				TempMin   float32 `json:"temp_min"`
				TempMax   float32 `json:"temp_max"`
			} `json:"main"`
		} `json:"list"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Группируем по дням
	dayData := make(map[string][]domain.Weather)

	for _, item := range result.List {
		date := item.Date[:10]
		dayData[date] = append(dayData[date], domain.Weather{
			Date:        date,
			Temp:        item.Temperature.Temp,
			TempMin:     item.Temperature.TempMin,
			TempMax:     item.Temperature.TempMax,
			Description: item.Weather[0].Description,
		})
	}

	var dates []string
	for date := range dayData {
		dates = append(dates, date)
	}

	sort.Strings(dates)

	var ls []domain.Weather
	for i := 0; i < count && i < len(dates); i++ {
		date := dates[i]
		records := dayData[date]

		min := records[0].TempMin
		max := records[0].TempMax

		for _, record := range records {
			if record.TempMin < min {
				min = record.TempMin
			}
			if record.TempMax > max {
				max = record.TempMax
			}
		}

		weather := domain.Weather{
			Date:        date,
			TempMin:     min,
			TempMax:     max,
			Description: records[0].Description,
		}

		if len(ls) == count {
			break
		}

		ls = append(ls, weather)
	}

	return ls, nil
}
