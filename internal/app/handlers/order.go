package handlers

import (
	"encoding/json"
	"errors"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"io"
	"log"
	"net/http"
	"strconv"
)

// postOrder handles
// POST /api/user/orders — загрузка пользователем номера заказа для расчёта;
// 200 — номер заказа уже был загружен этим пользователем;
// 202 — новый номер заказа принят в обработку;
// 400 — неверный формат запроса;
// 401 — пользователь не аутентифицирован;
// 409 — номер заказа уже был загружен другим пользователем;
// 422 — неверный формат номера заказа;
// 500 — внутренняя ошибка сервера;
func (h *mainHandler) postOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		//take userID from context
		session := GetSession(r)
		//take order from body
		orderBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cant read order number", http.StatusUnprocessableEntity)
			return
		}
		orderStr := string(orderBytes)
		orderNum, err := strconv.ParseInt(orderStr, 10, 64)
		if err != nil {
			http.Error(w, "cant parse order number", http.StatusUnprocessableEntity)
			return
		}
		if !storage.OrderIsValid(orderNum) {
			http.Error(w, "order number is invalid", http.StatusUnprocessableEntity)
			return
		}
		err = h.repository.CreateOrder(ctx, session.UserID, orderStr)
		if err != nil {
			log.Println("create order error", err)
			if errors.Is(err, storage.ErrOrderConflict) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if errors.Is(err, storage.ErrOrderDuplicate) {
				//номер заказа уже был загружен этим пользователем;
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusAccepted)
	}
}

// getOrders godoc
// @Summary			Получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
// @Description		get string by ID
// @ID				get-orders
// @Accept			json
// @Produce			json
// @Param			id path int true "Account ID"
// @Param			Cookie header object true "Auth token"
// @Success			200 {object} []storage.Order
// @Success			204 {object} []storage.Order  "нет данных для ответа"
// @Failure			401 {string} String "пользователь не авторизован"
// @Failure			500 {string} String "внутренняя ошибка сервера"
// @Router			/api/user/orders [get]
func (h *mainHandler) getOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//take userID from context
		session := GetSession(r)

		orders, err := h.repository.GetOrders(ctx, session.UserID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if len(orders) == 0 {
			//нет данных для ответа;
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		err = json.NewEncoder(w).Encode(orders)
		if err != nil {
			log.Println("marshal response error: ", err)
		}
	}
}
