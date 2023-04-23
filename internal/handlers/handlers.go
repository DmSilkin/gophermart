package handlers

import (
	"encoding/json"
	"errors"
	"internal/middleware"
	"internal/storage"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
)

type Controller struct {
	storage storage.StorageController // интерфейс для взаимодействия с БД
	logger  zerolog.Logger
	//
}

func NewController(storage storage.StorageController, logger zerolog.Logger) *Controller {
	return &Controller{
		storage: storage,
		logger:  logger,
	}
}

func (c Controller) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.GzipHandle, middleware.UnGzipHandle, middleware.CheckCookieHandle)

	r.Get("/", c.defaultEndpointHandler)
	r.Get("/api/user/orders", c.userGetOrdersHandler)
	r.Get("/api/user/balance", c.userBalanceHandler)
	r.Get("/api/user/withdrawals", c.userWithdrawalsHandler)

	r.Post("/api/user/register", c.userRegisterHandler)
	r.Post("/api/user/login", c.userLoginHandler)
	r.Post("/api/user/orders", c.userPostOrdersHandler)
	r.Post("/api/user/balance/withdraw", c.userPostWithDrawBalanceHandler)

	return r
}

func (c Controller) defaultEndpointHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("Welcome to GOPHERMART!"))
	rw.Header().Set("Content-Type", "application/json")
}

func (c Controller) userGetOrdersHandler(rw http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("gophermartCookie")

	exist, _ := c.storage.IsUserExist(cookie.Value)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	orders, err := c.storage.GetOrders(cookie.Value)

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

func (c Controller) userBalanceHandler(rw http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("gophermartCookie")

	exist, _ := c.storage.IsUserExist(cookie.Value)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	userBalance, err := c.storage.GetBalance(cookie.Value)

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

func (c Controller) userWithdrawalsHandler(rw http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("gophermartCookie")

	exist, _ := c.storage.IsUserExist(cookie.Value)

	if !exist {
		http.Error(rw, "User does not exist!", http.StatusUnauthorized)
		return
	}

	withdrawals, err := c.storage.GetWithdrawals(cookie.Value)

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

func (c Controller) userRegisterHandler(rw http.ResponseWriter, r *http.Request) {
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
	rw.Header().Add("Authorization", `Basic realm="Give username and password"`)
}

func (c Controller) userLoginHandler(rw http.ResponseWriter, r *http.Request) {
	var userInfo storage.UserInfo
	if err := json.NewDecoder(r.Body).Decode(&userInfo); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	err := c.storage.IsUserValid(userInfo)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}

	cookie := createCookieForUser(userInfo.Login)
	http.SetCookie(rw, &cookie)
	rw.Header().Add("Authorization", `Basic realm="Give username and password"`)
}

func (c Controller) userPostOrdersHandler(rw http.ResponseWriter, r *http.Request) {
	content := r.Header.Get("Content-Type")
	if content != "text/plain" && content != "" {
		http.Error(rw, "content-type not supported!", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("gophermartCookie")

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

	orderCode, err := c.storage.AddOrder(cookie.Value, string(requestData))

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

func (c Controller) userPostWithDrawBalanceHandler(rw http.ResponseWriter, r *http.Request) {
	content := r.Header.Get("Content-Type")
	if content != "application/json" && content != "" {
		http.Error(rw, "content-type not supported!", http.StatusBadRequest)
		return
	}
	cookie, _ := r.Cookie("gophermartCookie")

	exist, _ := c.storage.IsUserExist(cookie.Value)

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

	err = c.storage.WithdrawBalance(cookie.Value, withdrawal)

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

func createCookieForUser(login string) http.Cookie {
	return http.Cookie{
		Name:     "gophermartCookie",
		Value:    login,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
