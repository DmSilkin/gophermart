package accrual

import (
	"encoding/json"
	"fmt"
	"internal/storage"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
)

type AccrualController struct {
	storage        *storage.DBController
	logger         zerolog.Logger
	accrualAddress string
	client         *resty.Client
}

func NewAccrualController(storage *storage.DBController, logger zerolog.Logger, accrualAddress string) (*AccrualController, error) {
	client := resty.New().
		SetBaseURL(accrualAddress).
		SetRetryCount(100).
		SetRetryWaitTime(10 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second)

	return &AccrualController{
		storage:        storage,
		logger:         logger,
		accrualAddress: accrualAddress,
		client:         client,
	}, nil
}

func (a AccrualController) UpdateAccrualInfo() error {
	a.logger.Trace().Msg("UpdateAccrualInfo()")

	orders, err := a.storage.GetOrdersForAccrual()

	if err != nil {
		return err
	}

	for _, order := range orders.Orders {
		var orderFromAccrual storage.Order
		response, err := a.client.R().
			Get(a.client.BaseURL + "/api/orders/" + order.Number)

		if err != nil {
			a.logger.Error().Msg(err.Error())
		}

		err = json.Unmarshal(response.Body(), &orderFromAccrual)

		if err != nil {
			a.logger.Error().Msg(err.Error())
		}
		fmt.Println("got order:", orderFromAccrual)

		if orderFromAccrual.Status != "REGISTERED" { // если статус заказа в аккруале REGISTERED, то ничего не делаем
			err := a.storage.UpdateOrderInfo(order.Number, orderFromAccrual.Status, orderFromAccrual.Accrual)
			if err != nil {
				a.logger.Error().Msg(err.Error())
				break
			}

			if orderFromAccrual.Accrual != nil { // если от аккруала пришло зачисление баллов по заказу, то обновляем баланс пользователя
				err := a.storage.UpdateBalanceInfo(order.Number, orderFromAccrual.Accrual)
				if err != nil {
					a.logger.Error().Msg(err.Error())
					break
				}
			}
		}
	}

	return nil
}
