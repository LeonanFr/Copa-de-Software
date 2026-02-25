package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"copasoftware/internal/config"
	"copasoftware/internal/database"
	"copasoftware/internal/http"
)

func main() {
	cfg := config.Load()

	mongoDB, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("falha ao conectar no MongoDB: ", err)
	}
	defer func() {
		if err := mongoDB.Disconnect(); err != nil {
			log.Println("erro ao desconectar do MongoDB:", err)
		}
	}()

	router := http.NewRouter(mongoDB)
	srv := http.New(cfg.ServerPort, router)

	go func() {
		log.Printf("servidor iniciado na porta %s", cfg.ServerPort)
		if err := srv.Start(); err != nil {
			log.Fatal("erro ao iniciar servidor: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("desligando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("erro no graceful shutdown: ", err)
	}
}
