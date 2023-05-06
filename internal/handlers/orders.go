package handlers

import (
	"encoding/json"
	"internal/storage"
	"io/ioutil"
	"net/http"
)

func (c Controller) UserGetOrdersHandler(rw http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("Authorization")

	exist, _ := c.storage.IsUserExist(username)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	orders, err := c.storage.GetOrders(username)

	if err != nil {
		http.Error(rw, "server error", http.StatusInternalServerError)
		return
	}

	if len(orders.Orders) == 0 {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.Marshal(orders.Orders)
	rw.Header().Set("Content-Type", "application/json")
	if err == nil {
		rw.Write([]byte(body))
	} else {
		http.Error(rw, "server error", http.StatusInternalServerError)
	}
}

func (c Controller) UserPostOrdersHandler(rw http.ResponseWriter, r *http.Request) {
	content := r.Header.Get("Content-Type")
	if content != "text/plain" && content != "" {
		http.Error(rw, "content-type not supported!", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("Authorization")

	requestData, err := ioutil.ReadAll(r.Body)
	c.logger.Info().Msg(string(requestData))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	err = storage.IsOrderNumberValid(string(requestData))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	orderCode, err := c.storage.AddOrder(username, string(requestData))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	switch orderCode {
	case storage.ADDED:
		rw.WriteHeader(http.StatusAccepted)
		return
	case storage.ALREADY_MADE_BY_USER:
		rw.WriteHeader(http.StatusOK)
		return
	case storage.ALREADY_MADE_BY_ANOTHER_USER:
		http.Error(rw, "Order already made by another user", http.StatusConflict)
		return
	}

}
