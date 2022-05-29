package handlers

import (
	"encoding/json"
	"net/http"
)

type login struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// postRegister handles
// POST /api/user/register - регистрация пользователя;
// 200 - пользователь успешно зарегистрирован и аутентифицирован;
// 400 - неверный формат запроса;
// 409 - логин уже занят;
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) postRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var loginData login
		if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		userID, err := h.Repository.CreateUser(ctx, loginData.Login, loginData.Password)
		if err != nil {
			//TODO: 409, 500
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		session, err := h.Repository.CreateSession(ctx, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cookie := &http.Cookie{
			Path:    "/",
			Name:    "auth",
			Value:   session.Token,
			Expires: session.ExpiresAt,
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusOK)
	}
}

// postLogin handles
// POST /api/user/login - аутентификация пользователя;
// 200 - пользователь успешно аутентифицирован;
// 400 - неверный формат запроса;
// 401 - неверная пара логин/пароль;
// 500 - внутренняя ошибка сервера.
func (h *MainHandler) postLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
