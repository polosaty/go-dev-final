package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/polosaty/go-dev-final/internal/app/storage"
	"log"
	"net/http"
	"strconv"
	"time"
)

type OrderChecker struct {
	db                   storage.Repository
	orderCheckChan       chan storage.OrderForCheckStatus
	orderUpdateChan      chan storage.OrderUpdateStatus
	accrualSystemAddress string
}

func NewOrderChecker(db storage.Repository, accrualSystemAddress string) *OrderChecker {
	return &OrderChecker{
		db:                   db,
		orderCheckChan:       make(chan storage.OrderForCheckStatus, 10),
		orderUpdateChan:      make(chan storage.OrderUpdateStatus, 10),
		accrualSystemAddress: accrualSystemAddress,
	}
}

func (c *OrderChecker) stop() {
	close(c.orderCheckChan)
}

func (c *OrderChecker) SelectOrders(ctx context.Context, limit int) {
	if limit == 0 {
		limit = 100
	}

	go c.saveOrderStatuses(ctx) // stops by context
	go c.CheckOrders()          // stops by closing chanel

	var uploadedAfter *time.Time = nil

	//выбираем ордеры "по кругу": если вернулось меньше лимита, то начинаем с начала
	//если ничего не выбралось, при uploadedAfter == nil, спим (TODO: возможно до сигнала)
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
					time.Sleep(time.Millisecond * 500)
				} else {
					uploadedAfter = nil
				}
			} else {
				uploadedAfter = &orders[len(orders)-1].UploadedAt
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
	defer close(c.orderUpdateChan)

	for order := range c.orderCheckChan {
		log.Println("check order status: ", order.OrderNum)
		orderStatus, err := c.CheckOrder(order.OrderNum)
		if err != nil {
			log.Println("cant get status for order", err)
			continue
		}
		log.Println("got order status: ", order.OrderNum, order.Status)
		c.orderUpdateChan <- *orderStatus
	}
}

var ErrOrderStatusNotReady = errors.New("order result not ready")

func (c *OrderChecker) CheckOrder(order string) (*storage.OrderUpdateStatus, error) {
	var result storage.OrderUpdateStatus
	client := resty.New().
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if err != nil {
				log.Println("resty get order status error: ", err)
			}

			if r.StatusCode() == http.StatusTooManyRequests {
				//read Retry-After: N and sleep N seconds
				retryAfter := r.Header().Get("Retry-After")
				var retryAfterInt int64
				retryAfterInt, err = strconv.ParseInt(retryAfter, 10, 64)
				if err != nil {
					log.Println("resty parse header Retry-After error: ", err)
					retryAfterInt = 1
				}
				time.Sleep(time.Duration(retryAfterInt) * time.Second)

			}
			return r.StatusCode() != http.StatusOK && r.StatusCode() != http.StatusNoContent

		}).
		SetRetryCount(2).
		SetBaseURL(c.accrualSystemAddress).
		SetDoNotParseResponse(true)

	resp, err := client.R().Get("/api/orders/" + order)

	if err != nil {
		return nil, fmt.Errorf("cant get order status %w", err)
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil, ErrOrderStatusNotReady
	}

	if err = json.NewDecoder(resp.RawResponse.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("resty parse response error: %w", err)
	}

	if result.Status != "INVALID" && result.Status != "PROCESSED" {
		return nil, ErrOrderStatusNotReady
	}
	result.ProcessedAt = time.Now()
	return &result, nil
}

func (c *OrderChecker) saveOrderStatuses(ctx context.Context) {
	buffLen := 10
	//накапливаем статусы чтобы одним запросом обновить
	statuses := make([]storage.OrderUpdateStatus, 0, buffLen*2) // *2 чтобы не ресайзить в случае ошибок
	//если не набирается полный slice статусов, то по таймауту сбрасываем сколько есть
	ticker := time.NewTicker(time.Millisecond * 500)
L:
	for {
		select {
		case <-ctx.Done():
			break L
		case status := <-c.orderUpdateChan:
			if status.Status != "INVALID" && status.Status != "PROCESSED" {
				log.Printf("wrong status to save: %v\n", status)
			} else {
				statuses = append(statuses, status)
			}

			if len(statuses) == buffLen {
				if err := c.db.UpdateOrderStatus(ctx, statuses); err != nil {
					log.Println("save statuses error: ", err)
					continue
				}
				statuses = statuses[:0]
			}
		case <-ticker.C:
			if len(statuses) < 1 {
				continue
			}
			if err := c.db.UpdateOrderStatus(ctx, statuses); err != nil {
				log.Println("save statuses error: ", err)
				continue
			}
			statuses = statuses[:0]
		}
	}

	for status := range c.orderUpdateChan {
		statuses = append(statuses, status)
	}
	if err := c.db.UpdateOrderStatus(ctx, statuses); err != nil {
		log.Println("save statuses error: ", err)
	}
}
