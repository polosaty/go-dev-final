package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/polosaty/go-dev-final/internal/app/storage"

	_ "github.com/polosaty/go-dev-final/internal/app/docs"
	"github.com/swaggo/http-swagger"
)

type mainHandler struct {
	chiMux     *chi.Mux
	repository storage.Repository
}

// @title Swagger Example API
// @version 1.0
// @description Это учебный проект по курсу go-разработчкик.

// @host localhost:8080
// @BasePath /
// @query.collection.format multi

// @securityDefinitions.apikey ApiKeyAuth
// @in cookie
// @name auth

// @securitydefinitions.oauth2.application OAuth2Application
// @tokenUrl https://example.com/oauth/token
// @scope.write Grants write access
// @scope.admin Grants read and write access to administrative information

// @securitydefinitions.oauth2.implicit OAuth2Implicit
// @authorizationurl /api/user/login
// @scope.write Grants write access
// @scope.admin Grants read and write access to administrative information

// @securitydefinitions.oauth2.password OAuth2Password
// @tokenUrl https://example.com/oauth/token
// @scope.read Grants read access
// @scope.write Grants write access
// @scope.admin Grants read and write access to administrative information

// @securitydefinitions.oauth2.accessCode OAuth2AccessCode
// @tokenUrl https://example.com/oauth/token
// @authorizationurl https://example.com/oauth/authorize
// @scope.admin Grants read and write access to administrative information

// @x-extension-openapi {"example": "value on a json format"}
func NewMainHandler(repository storage.Repository) *chi.Mux {

	h := &mainHandler{chiMux: chi.NewMux(), repository: repository}
	h.chiMux.Use(gzipInput)
	h.chiMux.Use(gzipOutput)
	h.chiMux.Use(middleware.RequestID)
	h.chiMux.Use(middleware.RealIP)
	h.chiMux.Use(middleware.Logger)
	h.chiMux.Use(middleware.Recoverer)

	h.chiMux.Get("/docs/*", httpSwagger.Handler())

	h.chiMux.Route("/api/user", func(r chi.Router) {
		r.Post("/register", h.postRegister())
		r.Post("/login", h.postLogin())

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(repository))

			r.Post("/orders", h.postOrder())
			r.Get("/orders", h.getOrders())
			r.Route("/balance", func(r chi.Router) {
				r.Get("/", h.getBalance())
				r.Post("/withdraw", h.postWithdrawal())
				r.Get("/withdraws", h.getWithdraws())
			})
		})

	})

	return h.chiMux
}
