package store

import (
	"slices"
	"sync"
	"weather/internal/domain"
)

// Store хранит все данные приложения в памяти
type Store struct {
	mu sync.RWMutex

	// users хранит пользователей по их ID
	users map[int64]*domain.User

	// usersByLogin хранит пользователей по логину
	usersByLogin map[string]*domain.User

	// cities хранит любимые города пользователя
	cities map[int64][]*domain.City

	// nextID используется для генерации уникальных числовых ID
	nextID int64
}

func New() *Store {
	return &Store{
		mu:           sync.RWMutex{},
		users:        make(map[int64]*domain.User),
		usersByLogin: make(map[string]*domain.User),
		cities:       make(map[int64][]*domain.City),
		nextID:       1,
	}
}

// CreateUser добавляет нового пользователя
// Возвращает domain.ErrUserExists если логин уже занят
func (s *Store) CreateUser(login, passwordHash string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.usersByLogin[login]; ok {
		return nil, domain.ErrUserExists
	}

	NewUser := &domain.User{
		ID:           s.nextID,
		Login:        login,
		PasswordHash: passwordHash,
	}

	s.users[s.nextID] = NewUser
	s.usersByLogin[login] = NewUser
	s.nextID += 1

	return NewUser, nil
}

// GetUserByLogin возвращает пользователя по логину
// Возвращает domain.ErrUserNotFound если пользователь не найден
func (s *Store) GetUserByLogin(login string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if user, ok := s.usersByLogin[login]; ok {
		return user, nil
	}

	return nil, domain.ErrUserNotFound
}

// GetFavoriteCities возвращает избранные города пользователя
// Возвращает domain.ErrCitiesNotFound если пользователь не загрузил города
func (s *Store) GetFavoriteCities(userID int64) ([]*domain.City, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cities := s.cities[userID]
	if len(cities) > 0 {
		return cities, nil
	}

	return nil, domain.ErrCitiesNotFound
}

// AddCity добавляет город в избранное
// Service Возвращает ErrInvalidCityName если в названии есть цифры/специальные знаки
// Возвращает ErrCityExists если город уже добавлен
// Возвращает слайс из добавленных городов если имя пустое
func (s *Store) AddCity(userID int64, name string) ([]*domain.City, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cities := s.cities[userID]

	if name == "" {
		return cities, nil
	}

	if slices.ContainsFunc(cities, func(city *domain.City) bool {
		return city.Name == name
	}) {
		return nil, domain.ErrCityExists
	}

	NewCity := &domain.City{
		Name: name,
	}

	cities = append(cities, NewCity)
	s.cities[userID] = cities

	return cities, nil
}

// DeleteCity удаляет город из избранного
// Возвращет новый список с удаленным городом
func (s *Store) DeleteCity(userID int64, name string) ([]*domain.City, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cities := s.cities[userID]
	length := len(cities)

	// Удаляем элементы, у которых совпадает имя
	cities = slices.DeleteFunc(cities, func(city *domain.City) bool {
		return city.Name == name
	})

	if len(cities) == length {
		return nil, domain.ErrCityNotFound
	}

	s.cities[userID] = cities
	return cities, nil
}
