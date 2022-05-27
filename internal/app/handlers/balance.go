package handlers

import "net/http"

// getBalance handles
// GET /api/user/balance — получение текущего баланса счёта баллов лояльности пользователя;
func (h *MainHandler) getBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
