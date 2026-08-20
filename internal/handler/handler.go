package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"errors"
	"weather/internal/domain"
	"net/http"
	"log"
)

type Service interface {
	RegisterUser(login, password string) (string, error)
	LoginUser(login, password string) (string, error)
	AddFavoriteCity(userID int64, name string) ([]*domain.City, error)
	GetWeather(name string, days int) ([]domain.Weather, error)
	GetFavoriteInfo(userID int64, days int) (map[string][]domain.Weather, error)
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

type ErrorResponse struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

// WriteError записывает JSON-ответ с ошибкой
// Клиент видит только userMsg
func WriteError(w http.ResponseWriter, status int, code, userMsg string, internalErr error) {
	if internalErr != nil {
		log.Printf("ошибка code=%s status=%d: %v", code, status, internalErr)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Code: code,
		Message: userMsg,
	}

	json.NewEncoder(w).Encode(response)
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("WriteJSON: %v", err)
	}
}

type Req struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

// Register обрабатывает POST /api/user/register
// При успехе: 200 OK, заголовок Authorization с токеном
// При дублировании логина: 409 Confict
// При некорректных данных: 400 Bad Request
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	req := Req{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "неверный формат запроса", err)
		return
	}

	token, err := h.svc.RegisterUser(req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserExists):
			WriteError(w, http.StatusConflict, "USER_EXISTS", "пользователь уже существует", err)
		default:
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "внутрення ошибка", err)
		}
		return
	}

	w.Header().Set("Authorization", token)
	WriteJSON(w, http.StatusOK, nil)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req := Req{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "неверный формат запроса", err)
		return
	}

	token, err := h.svc.LoginUser(req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserExists):
			WriteError(w, http.StatusConflict, "USER_EXISTS", "пользователь уже существует", err)
		default:
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "внутрення ошибка", err)
		}
		return
	}

	w.Header().Set("Authorization", token)
	WriteJSON(w, http.StatusOK, nil)
}

type WeatherReq struct {
	Name string `json:"name"`
	Days int `json:"days"`
}

// GetWeather обрабатывает GET N
// При успехе: 200 OK
func (h *Handler) GetWeatherByName(w http.ResponseWriter, r *http.Request) {
	_, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "NO_USER", "не авторизован", nil)
		return
	}

	req := WeatherReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "WRONG_DATA", "неправильный тип данных", nil)
		return
	}

	spisok, err := h.svc.GetWeather(req.Name, req.Days)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "внутренняя ошибка", err)
	}

	text := ""
	for _, s := range spisok {
		text += fmt.Sprintf("%s: Min %v°C  Max %v°C Descriptio: %s\n", s.Date, s.TempMin, s.TempMax, s.Description)
	}
	
	WriteJSON(w, http.StatusOK, text)
}

type AddFavoriteCity struct {
	Name string `json:"name"`
}

// AddFavoriteCity обрабатывает POST /api/favorite
// При успехе: 200 OK
func (h *Handler) AddFavoriteCity(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "NO_USER", "не авторизован", nil)
		return
	}

	req := AddFavoriteCity{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_DATA", "неравильные данные", nil)
		return
	}

	cities, err := h.svc.AddFavoriteCity(userID, req.Name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "ошибка на сервере", err)
		return
	}

	Msg := DoText(cities)
	
	WriteJSON(w, http.StatusOK, Msg)
} 

// GetCityFavorite обрабатывает GET /api/favorite
// Возвращает список добавленных городов
// При успехе: 200 OK
func (h *Handler) GetCityFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "NO_USER", "не авторизован", nil)
		return
	}

	cities, err := h.svc.AddFavoriteCity(userID, "") 
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "внутрення ошибка", err)
		return
	}

	Msg := DoText(cities)

	WriteJSON(w, http.StatusOK, Msg)
}

type WeatherFavoriteRequest struct {
	Days int `json:"days"`
}

// GetWeatherFavorite обрабатывает GET /api/weather/favorite
// При успехе: 200 OK
func (h *Handler) GetWeatherFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "NO_USER", "не авторизован", nil)
		return
	}

	req := WeatherFavoriteRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_DATA", "неправильные данные", nil)
		return
	}

	weathers, err := h.svc.GetFavoriteInfo(userID, req.Days)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "внутренняя ошибка", err)
		return
	}

	msg := DoTextWeather(weathers)

	WriteJSON(w, http.StatusOK, msg)
}


// ---------------------------------------------------------------------------
// Вспомогательная функция для работы с текстом
// ---------------------------------------------------------------------------

// DoText - добавляет информацию к переданному тексту \n
func DoText(cities []*domain.City) string {
	text := "Успешно! Добавленные города:"
	for _, c := range cities {
		text += fmt.Sprintf("\n%s", c.Name)
	}
	return text
}

func DoTextWeather(weathers map[string][]domain.Weather) string {
	text := "Weather Info"
	for key, value := range weathers {
		text += fmt.Sprintf("\n[%s]:", key)

		for _, w := range value {
			text += fmt.Sprintf("%s  Min: %v°C  Max: %v°C  Description: %s", w.Date, w.TempMin, w.TempMax, w.Description)
		}
	}
	
	return text
}



// ---------------------------------------------------------------------------
// Вспомогательная функция для работы с контекстом
// ---------------------------------------------------------------------------

type contextKey string

const CtxKeyUserID contextKey = "userID"

// UserIDFromContext извлекает ID аутентифицированного пользователя
// Возвращает 0, false если значение отсутствует
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(CtxKeyUserID).(int64)
	if !ok || userID <= 0 {
		return 0, false
	}
	return userID, true
}