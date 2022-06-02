package server

import (
	"context"
	"github.com/polosaty/go-dev-final/internal/app/handlers"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"net/http"
)

func Serve(addr string, accrualSystemAddress string, db storage.Repository) error {
	handler := handlers.NewMainHandler(db)

	ctx := context.Background()
	orderChecker := NewOrderChecker(db)
	go orderChecker.SelectOrders(ctx, 10)
	// TODO: goroutine to check order statuses
	// TODO: goroutine to save order statuses

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server.ListenAndServe()
}
