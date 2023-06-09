package storage

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strconv"
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
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
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
	GetBalance(login string) (UserBalance, error)
	GetWithdrawals(login string) (WithDrawals, error)
	WithdrawBalance(login string, withdrawal WithDrawal) error
	//UpdateOrderStatus() //горутина, которая будет делать GET /api/orders/{number} для заказов, у которых статус NEW или PROCESSING
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

func (d *DBController) IsUserExist(login string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(ctx, "SELECT COUNT(*) FROM users WHERE login = $1", login)

	if err != nil {
		return false, err
	}

	defer rows.Close()

	var count int
	rows.Next()
	err = rows.Scan(&count)

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

	exist, err := d.IsUserExist(user.Login)

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
	exist, err := d.IsUserExist(user.Login)

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
	var userId int

	row := d.db.QueryRow("SELECT user_id FROM orders WHERE number = $1", number)

	if row.Scan(&userId) == sql.ErrNoRows {
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

func (d *DBController) GetOrders(login string) (Orders, error) {
	d.logger.Trace().Msg("GetOrders func!")
	orders := &Orders{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := d.getUserIdByLogin(login)
	d.logger.Debug().Int("User id is ", userId).Msg("")

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return Orders{}, err
	}

	rows, err := d.db.QueryContext(ctx, "SELECT number, status, accrual, uploaded_at from orders WHERE user_id = $1", userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return Orders{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(&o.Number, &o.Status, &o.Accrual, &o.UploadedAt)
		if err != nil {
			d.logger.Info().Err(err).Msg("")
			return Orders{}, err
		}

		orders.Orders = append(orders.Orders, o)
	}

	err = rows.Err()
	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return Orders{}, err
	}

	sortOrdersByTime(orders)

	return *orders, nil
}

func (d *DBController) GetBalance(login string) (UserBalance, error) {
	d.logger.Trace().Msg("GetBalance func!")
	userBalance := UserBalance{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := d.getUserIdByLogin(login)
	d.logger.Debug().Int("UserId", userId).Msg("")

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return UserBalance{}, err
	}

	rows, err := d.db.QueryContext(ctx, "SELECT current, withdrawn FROM balance WHERE user_id = $1", userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return UserBalance{}, err
	}

	defer rows.Close()

	rows.Next()
	err = rows.Scan(&userBalance.Current, &userBalance.Withdrawn)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return UserBalance{}, err
	}

	return userBalance, nil
}

func (d *DBController) GetWithdrawals(login string) (WithDrawals, error) {
	d.logger.Trace().Msg("GetWithdrawals func!")
	withdrawals := &WithDrawals{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err := d.getUserIdByLogin(login)
	d.logger.Debug().Int("UserId", userId).Msg("")

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return WithDrawals{}, err
	}

	rows, err := d.db.QueryContext(ctx, `SELECT orders.number, sum, processed_at from withdrawals 
											INNER JOIN orders ON withdrawals.order_id = orders.id
											WHERE withdrawals.user_id = $1`,
		userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return WithDrawals{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var w WithDrawal
		err = rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt)
		if err != nil {
			d.logger.Info().Err(err).Msg("")
			return WithDrawals{}, err
		}

		withdrawals.WithDrawals = append(withdrawals.WithDrawals, w)
	}

	err = rows.Err()
	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return WithDrawals{}, err
	}

	sortWithDrawalsByTime(withdrawals)

	return *withdrawals, nil
}

func (d *DBController) WithdrawBalance(login string, withdrawal WithDrawal) error {
	d.logger.Trace().Msg("WithdrawBalance func!")

	userId, err := d.getUserIdByLogin(login)
	d.logger.Debug().Int("UserId", userId).Msg("")

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	userBalance, err := d.GetBalance(login)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	if userBalance.Current < withdrawal.Sum {
		d.logger.Info().Err(ErrNotEnoughBalance).Msg(ErrNotEnoughBalance.Error())
		return ErrNotEnoughBalance
	}

	// транзакция: обновление баланса пользователя и добавление записи в withdrawals
	err = d.updateUserBalance(userId, userBalance, withdrawal)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	return nil
}

func (d *DBController) updateUserBalance(userId int, userBalance UserBalance, withdrawal WithDrawal) error {
	number, err := strconv.Atoi((withdrawal.Order))

	if err != nil {
		return err
	}

	orderId, err := d.getOrderIdByNumber(number)

	newBalance := UserBalance{
		Current:   userBalance.Current - withdrawal.Sum,
		Withdrawn: userBalance.Withdrawn + withdrawal.Sum,
	}

	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`UPDATE balance SET current=$1, withdrawn=$2 WHERE id=$3`)
	if err != nil {
		return err
	}

	if _, err = stmt.Exec(newBalance.Current, newBalance.Withdrawn, userId); err != nil {
		return err
	}

	stmt, err = tx.Prepare(`INSERT INTO withdrawals(user_id, order_id, sum, processed_at) VALUES($1,$2,$3,$4)`)
	if err != nil {
		return err
	}

	if _, err = stmt.Exec(userId, orderId, withdrawal.Sum, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}

	defer stmt.Close()

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (d *DBController) getOrderIdByNumber(number int) (int, error) {
	var orderId int
	row := d.db.QueryRow("SELECT id FROM orders WHERE number = $1", number)
	err := row.Scan(&orderId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return 0, err
	}

	return orderId, nil
}

func (d *DBController) getUserIdByLogin(login string) (int, error) {
	var userId int
	row := d.db.QueryRow("SELECT id FROM users WHERE login = $1", login)
	err := row.Scan(&userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return 0, err
	}

	return userId, nil
}

func sortOrdersByTime(orders *Orders) {
	sort.Slice(orders.Orders, func(i, j int) bool {
		return orders.Orders[i].UploadedAt.After(orders.Orders[j].UploadedAt)
	})
}

func sortWithDrawalsByTime(withdrawals *WithDrawals) {
	sort.Slice(withdrawals.WithDrawals, func(i, j int) bool {
		return withdrawals.WithDrawals[i].ProcessedAt.After(withdrawals.WithDrawals[j].ProcessedAt)
	})
}
