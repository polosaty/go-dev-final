package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// getBalance handles
// GET /api/user/balance — получение текущего баланса счёта баллов лояльности пользователя;
func (h *mainHandler) getBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//take userID from context
		session := GetSession(r)

		balance, err := h.repository.GetBalance(ctx, session.UserID)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(balance)
		if err != nil {
			log.Println("marshal response error: ", err)
		}
	}
}
