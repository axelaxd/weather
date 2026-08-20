package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"weather/internal/auth"
	"weather/internal/handler"
	"weather/internal/middleware"
	"weather/internal/request"
	"weather/internal/router"
	"weather/internal/service"
	"weather/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		text := "Warning: .env file not found"
        panic(text)
    }

	// Создаем все необходимое

	st := store.New()
	
	a := auth.New()

	req := request.New()

	svc := service.New(st, a, req)

	// Создаем обработчик и роутер (handler, router)
	hand := handler.New(svc)
	midl := middleware.New(a)
	rout := router.New(hand, midl)

	serviceChan := make(chan os.Signal, 1)
	signal.Notify(serviceChan, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr: ":8080",
		Handler: rout,
	}

	go func() {
		log.Printf("сервер запущен на порту :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	<-serviceChan
	log.Println("Получен сигнал завершения, останавливаем сервер...")

	// Graceful shutdown
	ctxShut, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	if err := server.Shutdown(ctxShut); err != nil {
		log.Printf("ошибка при остановке сервера: %v", err)
	}

	fmt.Println("Сервер остановлен")
}