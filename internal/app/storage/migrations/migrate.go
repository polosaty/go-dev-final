package migrations

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"log"
)

type DBInterface interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type migration func(ctx context.Context, db DBInterface) error

func Migrate(ctx context.Context, db DBInterface) error {
	_, err := db.Exec(
		ctx, `CREATE TABLE IF NOT EXISTS "revision" (version BIGSERIAL CONSTRAINT revision_version_pk PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("cannot get or create table revision: %w", err)
	}
	var version int
	err = db.QueryRow(
		ctx, "SELECT version FROM revision ORDER BY version DESC LIMIT 1").Scan(&version)

	if err != nil &&
		!(errors.Is(err, pgx.ErrNoRows)) {
		return fmt.Errorf("cannot get version: %w", err)
	}

	migrations := []migration{
		migration01,
	}

	for v, m := range migrations {
		if version < (v + 1) {
			log.Println("migrate database to version: ", v+1)
			if err = m(ctx, db); err != nil {
				return err
			}
		}
	}

	return nil
}
