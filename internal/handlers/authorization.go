package handlers

import (
	"encoding/json"
	"internal/storage"
	"net/http"
)

func (c Controller) UserRegisterHandler(rw http.ResponseWriter, r *http.Request) {
	var userInfo storage.UserInfo
	if err := json.NewDecoder(r.Body).Decode(&userInfo); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	err := c.storage.AddUser(userInfo)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusConflict)
		return
	}
	rw.Header().Add("Authorization", userInfo.Login)
}

func (c Controller) UserLoginHandler(rw http.ResponseWriter, r *http.Request) {

	var userInfo storage.UserInfo
	if err := json.NewDecoder(r.Body).Decode(&userInfo); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	err := c.storage.IsUserValid(userInfo)

	if err != nil {
		c.logger.Err(err).Msg("")
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	rw.Header().Add("Authorization", userInfo.Login)
}
