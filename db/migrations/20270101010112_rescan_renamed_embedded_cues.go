package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(func(ctx context.Context, tx *sql.Tx) error {
		return forceFullRescan(ctx, tx)
	}, func(context.Context, *sql.Tx) error {
		return nil
	})
}
