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
		var pge *pgconn.PgError
		if errors.As(err, &pge) && pge.SQLState() == "23505" {

			var orderUserID int64
			selErr := s.db.QueryRow(ctx, `SELECT "user_id" FROM "order" WHERE "order" = $1`, order).
				Scan(&orderUserID)

			if selErr != nil {
				return fmt.Errorf("cant select duclicate order: %w", selErr)
			}

			if orderUserID == userID {
				// одрер уже есть - пользователь тот же -> already uploaded 200
				return ErrOrderDuplicate
			} else {
				// одрер уже есть - пользователь другой -> conflict 409
				return ErrOrderConflict
			}

		}
		return fmt.Errorf("create order error: %w", err)
	}

	return nil
}

func (s *PG) GetOrders(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := s.db.Query(ctx,
		`SELECT "order", "accrual", "status", "processed_at", "uploaded_at" 
		FROM "order" WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("cant select orders: %w", err)
	}
	var orders []Order

	for rows.Next() {
		var v Order
		err = rows.Scan(&v.OrderNum, &v.Accrual, &v.Status, &v.processedAt, &v.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("cant parse row from select orders: %w", err)
		}
		orders = append(orders, v)
	}
	return orders, nil

}

func (s *PG) GetBalance(ctx context.Context, userID int64) (*Balance, error) {
	balance := &Balance{}
	err := s.db.QueryRow(ctx,
		`SELECT balance, withdrawn FROM  "user" WHERE id = $1`, userID).
		Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (s *PG) CreateWithdrawal(ctx context.Context, userID int64, withdrawal Withdrawal) error {
	//под транзакцией
	// - вычесть сумму из баланса пользователя и добавить сумму в списания пользователя
	// - если баланс окажется меньше 0 откатить транзакцию
	// - зарегистрировать списание

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx error: %w", err)
	}
	var newBalance float64
	err = s.db.QueryRow(ctx,
		`UPDATE "user" SET balance = balance - $1, withdrawn = withdrawn + $1 WHERE id = $2 
         RETURNING balance`,
		withdrawal.Sum, userID).
		Scan(&newBalance)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("update user balance error: %w", err)
	}
	if newBalance < 0 {
		err = tx.Rollback(ctx)
		if err != nil {
			return fmt.Errorf("rollback update user balance error: %w", err)
		}

		return ErrInsufficientBalance

	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO "withdrawal"("order", "sum", "user_id", "processed_at") VALUES($1, $2, $3, now())`,
		withdrawal.OrderNum, withdrawal.Sum, userID)

	if err != nil {
		return fmt.Errorf("create withdrawal error: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("cant commit tx %w", err)
	}

	return nil

}

func (s *PG) GetWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error) {
	rows, err := s.db.Query(ctx,
		`SELECT "order", "sum", "processed_at" 
		FROM "withdrawal" WHERE "user_id" = $1 ORDER BY "processed_at" ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("cant select orders: %w", err)
	}
	var withdrawals []Withdrawal

	for rows.Next() {
		var v Withdrawal
		err = rows.Scan(&v.OrderNum, &v.Sum, &v.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("cant parse row from select withdrawals: %w", err)
		}
		withdrawals = append(withdrawals, v)
	}
	return withdrawals, nil
}
