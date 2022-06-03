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
		FROM "order" WHERE user_id = $1 ORDER BY "uploaded_at"`, userID)
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

func (s *PG) SelectOrdersForCheckStatus(ctx context.Context, limit int, uploadedAfter *time.Time) ([]OrderForCheckStatus, error) {

	var (
		queryParams []interface{}
		query       string
	)
	if uploadedAfter != nil {
		query = `SELECT "order", "status", "uploaded_at" FROM "order" ` +
			` WHERE "status" NOT IN ('PROCESSED', 'INVALID') ` +
			` AND "uploaded_at" > $1 ORDER BY "uploaded_at" FOR UPDATE SKIP LOCKED LIMIT $2`
		queryParams = append(queryParams, uploadedAfter)
	} else {
		query = `SELECT "order", "status", "uploaded_at" FROM "order" ` +
			` WHERE "status" NOT IN ('PROCESSED', 'INVALID') ` +
			` ORDER BY "uploaded_at" FOR UPDATE SKIP LOCKED LIMIT $1`
	}
	queryParams = append(queryParams, limit)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx error: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, query, queryParams...)

	if err != nil {
		return nil, fmt.Errorf("cant select orders: %w", err)
	}
	var orders []OrderForCheckStatus
	ordersInRegisteredStatus := make([]string, 0, limit)
	for rows.Next() {
		var v OrderForCheckStatus
		err = rows.Scan(&v.OrderNum, &v.Status, &v.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("cant parse row from select orders: %w", err)
		}
		orders = append(orders, v)
		if v.Status == "NEW" {
			ordersInRegisteredStatus = append(ordersInRegisteredStatus, v.OrderNum)
		}
	}

	if len(ordersInRegisteredStatus) > 0 {
		_, err = tx.Exec(ctx,
			`UPDATE "order" SET "status" = 'PROCESSING' WHERE "order" = ANY($1)`, ordersInRegisteredStatus)
		if err != nil {
			return nil, fmt.Errorf("cant change orders status from NEW to PROCESSING: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("cant commit change orders status from NEW to PROCESSING: %w", err)
		}

	}

	return orders, nil
}

func (s *PG) UpdateOrderStatus(ctx context.Context, orders []OrderUpdateStatus) error {

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx,
		`CREATE TEMP TABLE tmp_table ON COMMIT DROP AS `+
			` SELECT "order", "status", "accrual", "processed_at"  FROM "order" WITH NO DATA`)
	if err != nil {
		return fmt.Errorf("cannot create temp table: %w", err)
	}
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"tmp_table"},
		[]string{"order", "status", "accrual", "processed_at"},
		pgx.CopyFromSlice(len(orders), func(i int) ([]interface{}, error) {
			row := orders[i]
			return []interface{}{row.OrderNum, row.Status, row.Accrual, row.ProcessedAt}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("cannot insert rows to temp table: %w", err)
	}

	_, err = tx.Exec(ctx,
		`WITH last_status as ( `+
			` SELECT * FROM tmp_table `+
			` WHERE NOT EXISTS(`+
			`  SELECT 1 FROM tmp_table following WHERE tmp_table.processed_at < following.processed_at)), `+
			`updates as (`+
			` UPDATE "order" SET `+
			`  "status" = last_status.status, `+
			`  "processed_at" = last_status.processed_at, `+
			`  "accrual" = last_status.accrual `+
			` FROM last_status WHERE last_status.order = "order"."order" AND "order".status != last_status.status `+
			` RETURNING "user_id", last_status."accrual", last_status."status"), `+
			`grouped_updates as ( `+
			` SELECT sum(accrual) AS accrual_sum, user_id `+
			`  FROM updates `+
			`  WHERE status = 'PROCESSED' `+
			`  GROUP BY updates.user_id) `+
			`UPDATE "user" `+
			` SET balance = balance + accrual_sum `+
			` FROM grouped_updates `+
			` WHERE "user"."id" = grouped_updates.user_id`)
	if err != nil {
		return fmt.Errorf("cannot update order from temp table: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
