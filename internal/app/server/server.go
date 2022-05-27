package server

import (
	"github.com/polosaty/go-dev-final/internal/app/handlers"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"net/http"
)

func Serve(addr string, accrualSystemAddress string, db storage.Repository) error {
	//проверяем не забыт ли "/" в конце BASE_URL

	handler := handlers.NewMainHandler(db)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server.ListenAndServe()
}
