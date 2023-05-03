package handlers

import (
	"encoding/json"
	"errors"
	"internal/storage"
	"net/http"
)

func (c Controller) UserBalanceHandler(rw http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("Authorization")

	exist, _ := c.storage.IsUserExist(username)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	userBalance, err := c.storage.GetBalance(username)

	if err != nil {
		http.Error(rw, "server error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(userBalance)
	rw.Header().Set("Content-Type", "application/json")
	if err == nil {
		rw.Write([]byte(body))
	} else {
		http.Error(rw, "server error", http.StatusInternalServerError)
	}
}

func (c Controller) UserWithdrawalsHandler(rw http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("Authorization")

	exist, _ := c.storage.IsUserExist(username)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	withdrawals, err := c.storage.GetWithdrawals(username)

	if err != nil {
		http.Error(rw, "server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals.WithDrawals) == 0 {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.Marshal(withdrawals)
	rw.Header().Set("Content-Type", "application/json")
	if err == nil {
		rw.Write([]byte(body))
	} else {
		http.Error(rw, "server error", http.StatusInternalServerError)
	}
}

func (c Controller) UserPostWithDrawBalanceHandler(rw http.ResponseWriter, r *http.Request) {
	content := r.Header.Get("Content-Type")
	if content != "application/json" && content != "" {
		http.Error(rw, "content-type not supported!", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("Authorization")

	exist, _ := c.storage.IsUserExist(username)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	var withdrawal storage.WithDrawal
	if err := json.NewDecoder(r.Body).Decode(&withdrawal); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	err := storage.IsOrderNumberValid(string(withdrawal.Order))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = c.storage.WithdrawBalance(username, withdrawal)

	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNotEnoughBalance):
			http.Error(rw, storage.ErrNotEnoughBalance.Error(), http.StatusPaymentRequired)
		default:
			http.Error(rw, "server error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
}
