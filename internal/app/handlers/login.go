package handlers

import "net/http"

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
	return func(w http.ResponseWriter, r *http.Request) {}
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
