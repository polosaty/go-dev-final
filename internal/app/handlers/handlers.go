package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/polosaty/go-dev-final/internal/app/storage"
)

type MainHandler struct {
	*chi.Mux
	Repository storage.Repository
}

func NewMainHandler(repository storage.Repository) *MainHandler {

	//var secretKey = []byte("secret key") // TODO: make random and save

	h := &MainHandler{Mux: chi.NewMux(), Repository: repository}
	//h.Use(gzipInput)
	//h.Use(gzipOutput)
	h.Use(middleware.RequestID)
	h.Use(middleware.RealIP)
	h.Use(middleware.Logger)
	h.Use(middleware.Recoverer)
	//h.Use(authMiddleware(secretKey))

	h.Route("/api/user", func(r chi.Router) {
		r.Post("/register", h.postRegister())
		r.Post("/login", h.postLogin())
		r.Post("/orders", h.postOrder())
		r.Get("/orders", h.getOrders())
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", h.getBalance())
			r.Get("/withdraw", h.postWithdrawal())
			r.Get("/withdraws", h.getWithdraws())
		})
	})

	return h
}
