package handlers

import "net/http"

// postOrder handles
// POST /api/user/orders — загрузка пользователем номера заказа для расчёта;
// 200 — номер заказа уже был загружен этим пользователем;
// 202 — новый номер заказа принят в обработку;
// 400 — неверный формат запроса;
// 401 — пользователь не аутентифицирован;
// 409 — номер заказа уже был загружен другим пользователем;
// 422 — неверный формат номера заказа;
// 500 — внутренняя ошибка сервера;
func (h *MainHandler) postOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

// getOrders handles
// GET /api/user/orders — получение списка загруженных пользователем номеров заказов,
// статусов их обработки и информации о начислениях;
// 204 — нет данных для ответа;
// 401 — пользователь не авторизован;
// 500 — внутренняя ошибка сервера;
func (h *MainHandler) getOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
