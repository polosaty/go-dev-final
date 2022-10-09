package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/polosaty/go-dev-final/internal/app/handlers"
	"github.com/polosaty/go-dev-final/internal/app/storage"
)

func Serve(ctx context.Context, addr string, accrualSystemAddress string, db storage.Repository) error {
	handler := handlers.NewMainHandler(db)

	var wg sync.WaitGroup
	orderCheckerCtx, cancelOrderCheckerCtx := context.WithCancel(ctx)
	orderChecker := NewOrderChecker(db, accrualSystemAddress)

	wg.Add(1)
	go orderChecker.SelectOrders(orderCheckerCtx, &wg, 10)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	gracefulShutdown := func() {
		log.Println("Shuting down server")
		if err := server.Shutdown(ctx); err != nil {
			log.Println("Shutdown server error: ", err)
		}
		log.Println("Cancel subCtx for orderChecker")
		cancelOrderCheckerCtx()
	}

	go func() {
		termChan := make(chan os.Signal, 1) // the channel used with signal.Notify should be buffered
		signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case signalValue := <-termChan: // Blocks here until either SIGINT or SIGTERM is received.
			log.Println("Received signal", signalValue)
			gracefulShutdown()
		case <-ctx.Done(): // Blocks here until context canceled.
			log.Println("Context canceled")
			gracefulShutdown()
		}

	}()

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		err = nil
		log.Println("Server closed")
	}
	wg.Wait()

	return err
}
