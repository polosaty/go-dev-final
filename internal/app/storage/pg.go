package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/polosaty/go-dev-final/internal/app/storage/migrations"
	"time"
)

type PG struct {
	Repository
	db dbInterface
}

type dbInterface interface {
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	Ping(context.Context) error
	Close()
}

func NewStoragePG(uri string) (*PG, error) {
	ctx := context.Background()
	conf, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to connect: parse dsn problem (dsn=%v): %w", uri, err)
	}

	//conf.MaxConns = 10
	conn, err := pgxpool.ConnectConfig(ctx, conf)

	if err != nil {
		return nil, fmt.Errorf("unable to connect to database(uri=%v): %w", uri, err)
	}

	repo := &PG{
		db: conn,
	}

	err = migrations.Migrate(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("can't apply migrations: %w", err)
	}

	return repo, nil
}

func (s *PG) CreateUser(ctx context.Context, login string, password string) (userID int64, err error) {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return
	}

	err = s.db.QueryRow(ctx,
		`INSERT INTO "user" (login, password) VALUES($1, $2)
			RETURNING id`, login, passwordHash).
		Scan(&userID)
	//TODO: handle already exists
	return
}

func (s *PG) LoginUser(ctx context.Context, login string, password string) (*Session, error) {
	var (
		userID       int64
		passwordHash string
	)
	err := s.db.QueryRow(ctx,
		`SELECT id, password FROM  "user" WHERE login = $1`, login).
		Scan(&userID, &passwordHash)
	if err != nil {
		return nil, err
	}
	if !CheckPasswordHash(password, passwordHash) {
		return nil, ErrWrongPassword
	}

	return s.CreateSession(ctx, userID)
}

func (s *PG) CreateSession(ctx context.Context, userID int64) (*Session, error) {
	session := &Session{
		Token:     generateToken(),
		ExpiresAt: time.Now().Add(time.Hour * 10),
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO user_session (user_id, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4)`,
		userID, session.Token, time.Now(), session.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session error: %w", err)
	}

	return session, nil
}

func (s *PG) GetUserByToken(ctx context.Context, token string) error {
	panic("implement me")
}

func (s *PG) CreateOrder(ctx context.Context, userID int64, order string) error {
	panic("implement me")
}

func (s *PG) GetOrders(ctx context.Context, userID int64) ([]Order, error) {
	panic("implement me")
}

func (s *PG) GetBalance(ctx context.Context, userID int64) (float64, error) {
	panic("implement me")
}

func (s *PG) CreateWithdrawal(ctx context.Context, userID int64, withdrawal Withdrawal) error {
	panic("implement me")
}

func (s *PG) GetWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error) {
	panic("implement me")
}
