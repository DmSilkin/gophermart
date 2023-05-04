package storage

import (
	"context"
	"database/sql"
	"sort"
	"strconv"
	"time"
)

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

	userId, err := d.GetUserIdByLogin(login)
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

func (d *DBController) GetOrdersForAccrual() (Orders, error) {
	d.logger.Trace().Msg("GetOrdersForAccrual func!")
	orders := &Orders{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.db.QueryContext(ctx, `SELECT number, status, accrual, uploaded_at from orders 
										 WHERE status = 'NEW' OR status = 'PROCESSING'`)

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

	return *orders, nil
}

func (d *DBController) UpdateOrderInfo(orderNumber string, newStatus string, accrual *float64) error {
	d.logger.Trace().Msg("UpdateOrderInfo func!")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orderId, err := d.getOrderIdByNumber(orderNumber)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	_, err = d.db.ExecContext(ctx, "UPDATE orders SET accrual=$1, status=$2 WHERE id=$3", accrual, newStatus, orderId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	return nil
}

func (d *DBController) getOrderIdByNumber(number string) (int, error) {
	var orderId int
	row := d.db.QueryRow("SELECT id FROM orders WHERE number = $1", number)
	err := row.Scan(&orderId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return 0, err
	}

	return orderId, nil
}

func (d *DBController) getUserIdByNumber(number string) (int, error) {
	var userId int
	row := d.db.QueryRow("SELECT user_id FROM orders WHERE number = $1", number)
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
