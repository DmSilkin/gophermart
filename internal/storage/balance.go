package storage

import (
	"context"
	"sort"
	"time"
)

func (d *DBController) GetBalance(userId int) (UserBalance, error) {
	d.logger.Trace().Msg("GetBalance func!")
	userBalance := UserBalance{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//userId, err := d.getUserIdByLogin(login)
	d.logger.Debug().Int("UserId", userId).Msg("")

	// if err != nil {
	// 	d.logger.Info().Err(err).Msg("")
	// 	return UserBalance{}, err
	// }

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

	userId, err := d.GetUserIdByLogin(login)
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

	userId, err := d.GetUserIdByLogin(login)
	d.logger.Debug().Int("UserId", userId).Msg("")

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	userBalance, err := d.GetBalance(userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	if userBalance.Current < withdrawal.Sum {
		d.logger.Info().Err(ErrNotEnoughBalance).Msg(ErrNotEnoughBalance.Error())
		return ErrNotEnoughBalance
	}

	// транзакция: обновление баланса пользователя и добавление записи в withdrawals
	err = d.withdrawUserBalance(userId, userBalance, withdrawal)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	return nil
}

func (d *DBController) UpdateBalanceInfo(number string, accrual *float64) error {

	userId, err := d.getUserIdByNumber(number)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}
	userBalance, err := d.GetBalance(userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	err = d.updateUserBalance(userId, userBalance, accrual)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	return nil
}

func (d *DBController) updateUserBalance(userId int, userBalance UserBalance, accrual *float64) error {
	newBalance := UserBalance{
		Current:   userBalance.Current + *accrual,
		Withdrawn: userBalance.Withdrawn,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.db.ExecContext(ctx, "UPDATE balance SET current=$1 where user_id=$2", newBalance.Current, userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return err
	}

	return nil
}

func (d *DBController) withdrawUserBalance(userId int, userBalance UserBalance, withdrawal WithDrawal) error {
	orderId, err := d.getOrderIdByNumber(withdrawal.Order)

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

func sortWithDrawalsByTime(withdrawals *WithDrawals) {
	sort.Slice(withdrawals.WithDrawals, func(i, j int) bool {
		return withdrawals.WithDrawals[i].ProcessedAt.After(withdrawals.WithDrawals[j].ProcessedAt)
	})
}
