package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
)

type OrderStatus int

const (
	NEW OrderStatus = iota + 1
	PROCESSING
	INVALID
	PROCESSED
)

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
	Number     int         `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
	UserId     int
}

type StorageController interface {
	IsUserExist(user UserInfo) (bool, error)
	IsUserValid(user UserInfo) error
	AddUser(user UserInfo) error
	AddOrder(login string, number string) (AddOrderReturn, error)
	//GetOrders(login string) Orders
	//GetBalance(login string)
	//WithdrawBalance(login string)
	//GetWithdrawals(login string)
	//UpdateOrderStatus() //горутина, которая будет делать GET /api/orders/{number} для заказов, у которых статус NEW или PROCESSING
}

type DBController struct {
	db *sql.DB // реализиует методы StorageController'a
}

func NewDBController(dsn string) (*DBController, error) {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	_, err = postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		return nil, err
	}

	return &DBController{
		db: db,
	}, nil
}

func (d *DBController) IsUserExist(user UserInfo) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(ctx, "SELECT COUNT(*) FROM users WHERE login = $1", user.Login)

	if err != nil {
		return false, err
	}

	defer rows.Close()

	var count int
	rows.Next()
	err = rows.Scan(&count)

	fmt.Println("IsUserExist count ", count)

	if err != nil {
		return false, err
	}

	err = rows.Err()
	if err != nil {
		return false, err
	}

	return count == 1, nil
}

func (d *DBController) IsUserValid(user UserInfo) error {

	exist, err := d.IsUserExist(user)

	if !exist || err != nil {
		return errors.New("Problem with user!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(ctx, "SELECT login, password FROM users WHERE login = $1", user.Login)

	if err != nil {
		return err
	}

	defer rows.Close()

	var userInfo UserInfo
	rows.Next()
	err = rows.Scan(&userInfo.Login, &userInfo.Password)

	if userInfo.Password != user.Password {
		return errors.New("Username or password wrong!")
	}

	return nil
}

func (d *DBController) AddUser(user UserInfo) error {

	exist, err := d.IsUserExist(user)

	if exist || err != nil {
		return errors.New("User already exist!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = d.db.ExecContext(ctx, `INSERT INTO users(login, password) VALUES($1,$2)`,
		user.Login, user.Password)

	if err != nil {
		return err
	}

	return nil
}

func (d *DBController) AddOrder(login string, number string) (AddOrderReturn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println("AddOrder()")
	var userId int

	n, _ := strconv.Atoi(number)

	row := d.db.QueryRow("SELECT user_id FROM orders WHERE number = $1", n)

	if row.Scan(&userId) == sql.ErrNoRows {
		fmt.Println("row.Err() == sql.ErrNoRows")
		row := d.db.QueryRow("SELECT id FROM users WHERE login = $1", login)
		err := row.Scan(&userId)

		if err != nil {
			return ERROR, err
		}

		_, err = d.db.ExecContext(ctx, `INSERT INTO orders(user_id, number, status, uploaded_at) VALUES($1,$2,$3,$4)`,
			strconv.Itoa(userId), number, "NEW", time.Now().Format(time.RFC3339))

		if err != nil {
			return ERROR, err
		}

		return ADDED, nil
	} else {
		fmt.Println("row.Err() != sql.ErrNoRows")
		fmt.Println("order.UserId = ", userId)

		var userLogin string
		row = d.db.QueryRow("SELECT login FROM users WHERE id = $1", userId)
		err := row.Scan(&userLogin)

		if err != nil {
			return ERROR, err
		}

		if userLogin != login {
			return ALREADY_MADE_BY_ANOTHER_USER, nil
		}

	}

	return ALREADY_MADE_BY_USER, nil
}
