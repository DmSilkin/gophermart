package storage

import (
	"context"
	"errors"
	"time"
)

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

func (d *DBController) GetUserIdByLogin(login string) (int, error) {
	var userId int
	row := d.db.QueryRow("SELECT id FROM users WHERE login = $1", login)
	err := row.Scan(&userId)

	if err != nil {
		d.logger.Info().Err(err).Msg("")
		return 0, err
	}

	return userId, nil
}
