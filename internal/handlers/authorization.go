package handlers

import (
	"encoding/json"
	"internal/storage"
	"net/http"
)

const COOKIE_NAME string = "gophermartCookie"

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
	cookie := createCookieForUser(userInfo.Login)
	http.SetCookie(rw, &cookie)
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

	cookie := createCookieForUser(userInfo.Login)
	http.SetCookie(rw, &cookie)
	rw.Header().Add("Authorization", userInfo.Login)
}

func createCookieForUser(login string) http.Cookie {
	return http.Cookie{
		Name:     COOKIE_NAME,
		Value:    login,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
