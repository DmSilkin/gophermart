package storage

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
)

var ErrNotEnoughBalance = errors.New("Current balance is not enough!")

type AddOrderReturn int

const (
	ALREADY_MADE_BY_USER AddOrderReturn = iota + 1
	ADDED
	ALREADY_MADE_BY_ANOTHER_USER
	ERROR
)

type UserInfo struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserBalance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Order struct {
	Number      string    `json:"number"`
	Status      string    `json:"status"`
	Accrual     *float64  `json:"accrual,omitempty"`
	UploadedAt  time.Time `json:"uploaded_at"`
	OrderNumber string    `json:"order"`
}

type Orders struct {
	Orders []Order `json:"orders"`
}

type WithDrawal struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
}

type WithDrawals struct {
	WithDrawals []WithDrawal `json:"withdrawals"`
}

type StorageController interface {
	IsUserExist(login string) (bool, error)
	IsUserValid(user UserInfo) error
	AddUser(user UserInfo) error
	AddOrder(login string, number string) (AddOrderReturn, error)
	GetOrders(login string) (Orders, error)
	GetBalance(userId int) (UserBalance, error)
	GetWithdrawals(login string) (WithDrawals, error)
	WithdrawBalance(login string, withdrawal WithDrawal) error
	UpdateOrderInfo(number string, newStatus string, accrual *float64) error
	UpdateBalanceInfo(number string, accrual *float64) error
	GetUserIdByLogin(login string) (int, error)
}

type DBController struct {
	db     *sql.DB // реализует методы StorageController'a
	logger zerolog.Logger
}

func NewDBController(dsn string, logger zerolog.Logger) (*DBController, error) {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://cmd/gophermart/migrations",
		"pgx", driver)

	if err != nil {
		return nil, err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, err
	}

	return &DBController{
		db:     db,
		logger: logger,
	}, nil
}
