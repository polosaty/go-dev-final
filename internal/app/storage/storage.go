package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type RFC3339DateTime struct {
	time.Time
}

func (c *RFC3339DateTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`) // remove quotes
	if s == "null" {
		return
	}
	c.Time, err = time.Parse(time.RFC3339, s)
	return
}

func (c RFC3339DateTime) MarshalJSON() ([]byte, error) {
	if c.Time.IsZero() {
		return nil, nil
	}
	return []byte(fmt.Sprintf(`"%s"`, c.Time.Format(time.RFC3339))), nil
}

type Order struct {
	OrderNum    string   `json:"number"`
	Status      string   `json:"status"`
	Accrual     *float64 `json:"accrual,omitempty"`
	processedAt *time.Time
	UploadedAt  *RFC3339DateTime `json:"uploaded_at"`
}

type OrderForCheckStatus struct {
	OrderNum   string
	Status     string
	UploadedAt time.Time
}

type OrderUpdateStatus struct {
	OrderNum    string  `json:"order"`
	Status      string  `json:"status"`
	Accrual     float64 `json:"accrual,omitempty"`
	ProcessedAt time.Time
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdrawal struct {
	OrderNum    string           `json:"order"`
	Sum         float64          `json:"sum"`
	ProcessedAt *RFC3339DateTime `json:"processed_at,omitempty"`
}

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, login string, password string) (int64, error)
	LoginUser(ctx context.Context, login string, password string) (*Session, error)
	CreateSession(ctx context.Context, userID int64) (*Session, error)
	GetUserByToken(ctx context.Context, token string) (int64, error)

	CreateOrder(ctx context.Context, userID int64, order string) error
	GetOrders(ctx context.Context, userID int64) ([]Order, error)

	GetBalance(ctx context.Context, userID int64) (*Balance, error)

	CreateWithdrawal(ctx context.Context, userID int64, withdrawal Withdrawal) error
	GetWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error)

	SelectOrdersForCheckStatus(ctx context.Context, limit int, uploadedAfter *time.Time) ([]OrderForCheckStatus, error)
	UpdateOrderStatus(ctx context.Context, orders []OrderUpdateStatus) error
}

var ErrWrongPassword = errors.New("wrong password")
var ErrWrongLogin = errors.New("wrong login")
var ErrDuplicateUser = errors.New("duplicate user")
var ErrWrongToken = errors.New("wrong token")

var ErrOrderDuplicate = errors.New("order already uploaded")
var ErrOrderConflict = errors.New("order conflict")

var ErrInsufficientBalance = errors.New("insufficient balance for withdrawn")

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken() string {
	return uuid.New().String()
}
