package storage

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Order struct {
	OrderNum    string   `json:"order"`
	Status      string   `json:"status"`
	Accrual     *float64 `json:"accrual,omitempty"`
	processedAt *time.Time
	UploadedAt  *time.Time `json:"uploaded_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdrawal struct {
	OrderNum    string     `json:"order"`
	Sum         float64    `json:"sum"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
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
