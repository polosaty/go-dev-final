package main

import (
	"context"
	"flag"
	"github.com/caarlos0/env/v6"
	"github.com/polosaty/go-dev-final/internal/app/config"
	"github.com/polosaty/go-dev-final/internal/app/server"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"log"
)

func main() {
	var cfg config.Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "server address")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address")
	flag.Parse()

	var db storage.Repository

	if db, err = storage.NewStoragePG(cfg.DatabaseURI); err != nil {
		log.Fatal(err)
	}
	log.Println("use postgres conn " + cfg.DatabaseURI + " as db")

	if err := server.Serve(context.Background(), cfg.RunAddress, cfg.AccrualSystemAddress, db); err != nil {
		log.Fatal("serve error:", err)
	}

}
