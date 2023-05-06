package handlers

import (
	"internal/accrual"
	"internal/config"
	"internal/middleware"
	"internal/storage"
	"log"
	"time"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

type Controller struct {
	storage storage.StorageController // интерфейс для взаимодействия с БД
	logger  zerolog.Logger
	accrual *accrual.AccrualController
}

func NewController(cfg config.ServerConfig, logger zerolog.Logger) *Controller {
	storage, err := storage.NewDBController(cfg.DatabaseURI, logger)

	if err != nil {
		log.Fatalln(err)
	}

	accrual, err := accrual.NewAccrualController(storage, logger, cfg.AccrualAddress)

	if err != nil {
		log.Fatalln(err)
	}

	return &Controller{
		storage: storage,
		logger:  logger,
		accrual: accrual,
	}
}

func (c Controller) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.GzipHandle, middleware.UnGzipHandle, middleware.CheckAuthorizationHandle)

	r.Get("/api/user/orders", c.UserGetOrdersHandler)
	r.Get("/api/user/balance", c.UserBalanceHandler)
	r.Get("/api/user/withdrawals", c.UserWithdrawalsHandler)

	r.Post("/api/user/register", c.UserRegisterHandler)
	r.Post("/api/user/login", c.UserLoginHandler)
	r.Post("/api/user/orders", c.UserPostOrdersHandler)
	r.Post("/api/user/balance/withdraw", c.UserPostWithDrawBalanceHandler)

	return r
}

func (c Controller) NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/", c.Router())

	return r
}

func (c Controller) ProcessAccrual(pollInterval time.Duration) error {
	tickerFlush := time.NewTicker(pollInterval)
	go func() {
		for {
			select {
			case <-tickerFlush.C:
				err := c.accrual.UpdateAccrualInfo()
				if err != nil {
					c.logger.Fatal().Err(err).Msg("Process Accrual error!")
				}
			}
		}
	}()

	return nil
}
