package storage

import (
	"context"
	"errors"
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

	//https://github.com/jackc/pgconn/issues/15#issuecomment-867082415
	var pge *pgconn.PgError
	if errors.As(err, &pge) {
		if pge.SQLState() == "23505" {
			// user already exists
			// Handle  duplicate key value violates
			return 0, ErrDuplicateUser
		}
		return 0, fmt.Errorf("create user error: %w", err)
	}

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
		if err == pgx.ErrNoRows {
			return nil, ErrWrongLogin
		}
		return nil, err
	}
	if !CheckPasswordHash(password, passwordHash) {
		return nil, ErrWrongPassword
	}

	return s.CreateSession(ctx, userID)
}

func (s *PG) CreateSession(ctx context.Context, userID int64) (*Session, error) {
	session := &Session{
		UserID:    userID,
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

func (s *PG) GetUserByToken(ctx context.Context, token string) (int64, error) {
	var userID int64

	err := s.db.QueryRow(ctx,
		`SELECT user_id FROM  "user_session" WHERE token = $1 and expires_at > now()`, token).
		Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ErrWrongToken
		}
		return 0, err
	}

	return userID, nil
}

func (s *PG) CreateOrder(ctx context.Context, userID int64, order string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO "order"("order", "user_id", "uploaded_at") VALUES($1, $2, $3)`,
		order, userID, time.Now())

	if err != nil {
		return fmt.Errorf("create order error: %w", err)
	}

	return nil
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
