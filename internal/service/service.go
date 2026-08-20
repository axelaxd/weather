package service

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"weather/internal/auth"
	"weather/internal/domain"
)

type Repository interface {
	CreateUser(login, passwordHash string) (*domain.User, error)
	GetUserByLogin(login string) (*domain.User, error)
	GetFavoriteCities(userID int64) ([]*domain.City, error)
	AddCity(userID int64, name string) ([]*domain.City, error)
	DeleteCity(userID int64, name string) ([]*domain.City, error)
}

type Requests interface {
	GetCityInfo(name string, count int) ([]domain.Weather, error)
}

// Service реализует бизнес-логику приложения
type Service struct {
	repo     Repository
	requests Requests
	auth     *auth.Auth
	mu       sync.RWMutex
}

// New создаёт Service
func New(repo Repository, a *auth.Auth, req Requests) *Service {
	return &Service{
		repo:     repo,
		requests: req,
		auth:     a,
		mu:       sync.RWMutex{},
	}
}

// ---------------------------------------------------------------------------
// Методы бизнес-логики
// ---------------------------------------------------------------------------

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func checkHash(password, passwordHash string) bool {
	hash := hashPassword(password)
	return hash == passwordHash
}

// RegisterUser регистрирует нового пользовтеля и возвращает токен аутентификации
func (s *Service) RegisterUser(login, password string) (string, error) {
	passwordHash := hashPassword(password)

	user, err := s.repo.CreateUser(login, passwordHash)
	if err != nil {
		return "", err
	}

	token, err := s.auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// LoginUser проверяет учётные данные и возвращет токен аутентификации
func (s *Service) LoginUser(login, password string) (string, error) {
	user, err := s.repo.GetUserByLogin(login)
	if err != nil {
		return "", err
	}

	if !checkHash(password, user.PasswordHash) {
		return "", domain.ErrInvalidCredentials
	}

	token, err := s.auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Добавляет избранный город
// Возвращает слайс из добавленных городов
// Если передать пустое имя, функция вернет уже добавленные
func (s *Service) AddFavoriteCity(userID int64, name string) ([]*domain.City, error) {
	return s.repo.AddCity(userID, name)
}

// Получает информацию о погоде
// на несколько дней
func (s *Service) GetWeather(name string, days int) ([]domain.Weather, error) {
	return s.requests.GetCityInfo(name, days)
}

// Возвращает мапу, где string - название города
// []domain.Weather - слайс по дням
func (s *Service) GetFavoriteInfo(userID int64, days int) (map[string][]domain.Weather, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cities, err := s.repo.GetFavoriteCities(userID)
	if err != nil {
		return nil, err
	}

	ls := make(map[string][]domain.Weather)
	for _, c := range cities {
		weather, err := s.GetWeather(c.Name, days)
		if err != nil {
			return nil, err
		}

		ls[c.Name] = weather
	}

	return ls, nil
}
