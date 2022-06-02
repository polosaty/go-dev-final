package server

import (
	"context"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"log"
	"time"
)

type OrderChecker struct {
	db              storage.Repository
	orderCheckChan  chan storage.OrderForCheckStatus
	orderUpdateChan chan storage.OrderUpdateStatus
}

func NewOrderChecker(db storage.Repository) *OrderChecker {
	return &OrderChecker{
		db:              db,
		orderCheckChan:  make(chan storage.OrderForCheckStatus, 10),
		orderUpdateChan: make(chan storage.OrderUpdateStatus, 10),
	}
}

func (c *OrderChecker) stop() {
	close(c.orderCheckChan)
}

func (c *OrderChecker) SelectOrders(ctx context.Context, limit int) {
	if limit == 0 {
		limit = 100
	}

	var uploadedAfter *time.Time = nil
	//выбираем ордеры по кругу если вернулось меньше лимита, то начинаем с начала
	//если ничего не выбралось, при uploadedAfter == nil, спим (возможно до сигнала)
	for {
		select {
		case <-ctx.Done():
			c.stop()
			return
		default:
			orders, err := c.db.SelectOrdersForCheckStatus(ctx, limit, uploadedAfter)
			if err != nil {
				log.Println("error selecting order from check status", err)
				continue
			}
			if len(orders) < limit {
				if uploadedAfter == nil {
					time.Sleep(2 * time.Second)
				} else {
					uploadedAfter = nil
				}
			}
			for _, order := range orders {
				select {
				case <-ctx.Done():
					c.stop()
					return
				default:
					c.orderCheckChan <- order
				}
			}

		}

	}

}

func (c *OrderChecker) CheckOrders() {
	for order := range c.orderCheckChan {
		orderStatus, err := c.CheckOrder(order.OrderNum)
		if err != nil {
			log.Println("cant get status for order", err)
			continue
		}
		c.orderUpdateChan <- orderStatus
	}
}

func (c *OrderChecker) CheckOrder(order string) (storage.OrderUpdateStatus, error) {
	panic("implement me")
}
