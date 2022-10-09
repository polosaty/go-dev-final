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
	"sync"
	"time"
)

var ErrOrderStatusNotReady = errors.New("order result not ready")

type OrderChecker struct {
	db                   storage.Repository
	accrualSystemAddress string
}

func NewOrderChecker(db storage.Repository, accrualSystemAddress string) *OrderChecker {
	return &OrderChecker{
		db:                   db,
		accrualSystemAddress: accrualSystemAddress,
	}
}

func (c *OrderChecker) SelectOrders(ctx context.Context, wg *sync.WaitGroup, limit int) {
	if limit == 0 {
		limit = 100
	}
	orderCheckChan := make(chan storage.OrderForCheckStatus, 10)

	wg.Add(1)
	go c.checkOrders(ctx, wg, orderCheckChan) // stops by closing chanel
	defer func() {
		close(orderCheckChan)
		wg.Done()
	}()

	var uploadedAfter *time.Time = nil

	//выбираем ордеры "по кругу": если вернулось меньше лимита, то начинаем с начала
	//если ничего не выбралось, при uploadedAfter == nil, спим (возможно до сигнала ctx.Done())
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.C:
			orders, err := c.db.SelectOrdersForCheckStatus(ctx, limit, uploadedAfter)
			if err != nil {
				log.Println("error selecting order from check status", err)
				continue
			}
			if len(orders) < limit {
				//значит дошли до конца и нужно начинать с первого одрера
				uploadedAfter = nil
			} else {
				//обновляем uploadedAfter из последнего выбранного ордера
				//для того чтобы в следующем запросе выбрать ордеры начиная со следующего
				uploadedAfter = &orders[len(orders)-1].UploadedAt
			}
			for _, order := range orders {
				select {
				case <-ctx.Done():
					return
				case orderCheckChan <- order:
					continue
				}
			}
			//if len(orders) == 0 {
			//	//значит мало ордеров на проверку
			//	time.Sleep(time.Second * 5)
			//}
		case <-ctx.Done():
			return
		}
	}

}

func (c *OrderChecker) checkOrders(
	ctx context.Context,
	wg *sync.WaitGroup,
	orderCheckChan <-chan storage.OrderForCheckStatus,
) {
	orderUpdateChan := make(chan *storage.OrderUpdateStatus, 10)

	wg.Add(1)
	go c.saveOrderStatuses(ctx, wg, orderUpdateChan) // stops by closing chanel

	defer func() {
		close(orderUpdateChan)
		wg.Done()
	}()

	for order := range orderCheckChan {
		log.Println("check order status: ", order.OrderNum)
		orderStatus, err := c.CheckOrder(order.OrderNum)
		if err != nil {
			log.Println("cant get status for order", err)
			continue
		}
		log.Println("got order status: ", orderStatus.OrderNum, orderStatus.Status)
		orderUpdateChan <- orderStatus
	}
}

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
func (c *OrderChecker) updateStatuses(ctx context.Context, statuses []storage.OrderUpdateStatus) error {
	if err := c.db.UpdateOrderStatus(ctx, statuses); err != nil {
		log.Println("save statuses error: ", err)
		return err
	}
	return nil
}

func (c *OrderChecker) saveOrderStatuses(
	ctx context.Context,
	wg *sync.WaitGroup,
	orderUpdateChan <-chan *storage.OrderUpdateStatus,
) {

	defer wg.Done()
	buffLen := 10
	//накапливаем статусы чтобы одним запросом обновить
	statuses := make([]storage.OrderUpdateStatus, 0, buffLen*2) // *2 чтобы не ресайзить в случае ошибок
	//если не набирается полный slice статусов, то по таймауту сбрасываем сколько есть
	ticker := time.NewTicker(time.Second * 5)
	saveCollectedStatuses := func(ctx context.Context) {
		if len(statuses) < 1 {
			return
		}

		if err := c.updateStatuses(ctx, statuses); err == nil {
			statuses = statuses[:0]
		}
	}
	for {
		select {
		case status := <-orderUpdateChan:
			if status == nil {
				return
			}
			if status.Status != "INVALID" && status.Status != "PROCESSED" {
				log.Printf("wrong status to save: %v\n", status)
			} else {
				statuses = append(statuses, *status)
			}

			if len(statuses) == buffLen {
				saveCollectedStatuses(ctx)
			}
		case <-ticker.C:
			saveCollectedStatuses(ctx)

		case <-ctx.Done():
			func() {
				ctxWithTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				saveCollectedStatuses(ctxWithTimeout)
			}()
			return
		}
	}

}
