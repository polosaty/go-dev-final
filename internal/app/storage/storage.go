package storage

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Order struct{}
type Withdrawal struct{}
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

	GetBalance(ctx context.Context, userID int64) (float64, error)

	CreateWithdrawal(ctx context.Context, userID int64, withdrawal Withdrawal) error
	GetWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error)
}

var ErrWrongPassword = errors.New("wrong password")
var ErrWrongLogin = errors.New("wrong login")
var ErrDuplicateUser = errors.New("duplicate user")
var ErrWrongToken = errors.New("wrong token")

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
