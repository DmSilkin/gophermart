package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"internal/config"
	"internal/handlers"

	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/rs/zerolog"
)

func main() {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	cfg, err := config.NewServerConfig()
	if err != nil {
		log.Fatalln(err)
	}
	logger := zerolog.New(os.Stdout).Level(1)

	controller := handlers.NewController(cfg, logger)

	r := chi.NewRouter()
	r.Mount("/", controller.Router())

	server := &http.Server{Addr: cfg.HTTPAddress, Handler: r}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-stop

	// горутина, которая получает заказы от аккруала с заданной периодичностью (по появлению заказа)
}
