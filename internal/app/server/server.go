package server

import (
	"context"
	"github.com/polosaty/go-dev-final/internal/app/handlers"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"net/http"
)

func Serve(addr string, accrualSystemAddress string, db storage.Repository) error {
	handler := handlers.NewMainHandler(db)

	ctx, cancel := context.WithCancel(context.Background())
	orderChecker := NewOrderChecker(db, accrualSystemAddress)
	go orderChecker.SelectOrders(ctx, 10)
	defer cancel()

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server.ListenAndServe()
}
