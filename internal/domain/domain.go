package domain

import (
	"errors"
)

type User struct {
	ID int64
	Login string
	PasswordHash string
}

type Weather struct {
	City string // Name
	Date string
	Main string
	Description string
	Temp float32
	FeelsLike float32
	TempMin float32
	TempMax float32 
}

type City struct {
	Name string
}

var (
	ErrUserExists = errors.New("user exists")
	ErrUserNotFound = errors.New("user not found")
	ErrCitiesNotFound = errors.New("cities not added")
	ErrInvalidCityName = errors.New("name of city must be A-Z")
	ErrCityExists = errors.New("city already exists")
	ErrCityNotFound = errors.New("city noy found")
	ErrInvalidCredentials = errors.New("invalid login or password")
)