package handlers

import (
	"encoding/json"
	"errors"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"log"
	"net/http"
	"strconv"
)

// postWithdrawal handles
// POST /api/user/balance/withdraw - запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа;
// 200 - успешная обработка запроса;
// 401 - пользователь не авторизован;
// 402 - на счету недостаточно средств;
// 422 - неверный номер заказа;
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) postWithdrawal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		//take userID from context
		session := GetSession(r)
		//take order from body
		var withdrawal storage.Withdrawal
		if err := json.NewDecoder(r.Body).Decode(&withdrawal); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		orderNum, err := strconv.ParseInt(withdrawal.OrderNum, 10, 64)
		if err != nil || !storage.OrderIsValid(orderNum) {
			http.Error(w, "order number is invalid", http.StatusUnprocessableEntity)
			return
		}
		err = h.Repository.CreateWithdrawal(ctx, session.UserID, withdrawal)
		if err != nil {
			log.Println("create withdrawal error", err)
			if errors.Is(err, storage.ErrInsufficientBalance) {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//успешная обработка запроса;
		w.WriteHeader(http.StatusOK)
	}
}

// getWithdraws handles
// GET /api/user/balance/withdrawals — получение информации о выводе средств с накопительного счёта пользователем.
// 204 - нет ни одного списания.
// 401 - пользователь не авторизован.
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) getWithdraws() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//take userID from context
		session := GetSession(r)

		orders, err := h.Repository.GetWithdrawals(ctx, session.UserID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if len(orders) == 0 {
			//нет ни одного списания;
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
