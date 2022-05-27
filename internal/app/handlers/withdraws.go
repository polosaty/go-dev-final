package handlers

import "net/http"

// postWithdrawal handles
// POST /api/user/balance/withdraw - запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа;
// 200 - успешная обработка запроса;
// 401 - пользователь не авторизован;
// 402 - на счету недостаточно средств;
// 422 - неверный номер заказа;
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) postWithdrawal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// getWithdraws handles
// GET /api/user/balance/withdrawals — получение информации о выводе средств с накопительного счёта пользователем.
// 204 - нет ни одного списания.
// 401 - пользователь не авторизован.
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) getWithdraws() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
