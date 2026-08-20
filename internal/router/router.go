package router

import (
	"net/http"
	"weather/internal/handler"
	"weather/internal/middleware"
)

func New(h *handler.Handler, m *middleware.Middleware) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/user/register", h.Register)
	mux.HandleFunc("POST /api/user/login", h.Login)

	// Которые требуют токен
	mux.Handle("GET /api/weather", m.Auth(http.HandlerFunc(h.GetWeatherByName)))
	mux.Handle("GET /api/favorite", m.Auth(http.HandlerFunc(h.GetCityFavorite)))
	mux.Handle("GET /api/weather/favorite", m.Auth(http.HandlerFunc(h.GetWeatherFavorite)))
	mux.Handle("POST /api/favorite", m.Auth(http.HandlerFunc(h.AddFavoriteCity)))

	return middleware.Logging(middleware.Recover(mux))
}
