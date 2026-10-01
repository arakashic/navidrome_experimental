package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(func(ctx context.Context, tx *sql.Tx) error {
		for _, column := range []string{"cue_start_sample", "cue_end_sample", "cue_source_rate"} {
			if _, err := tx.ExecContext(ctx, "ALTER TABLE media_file ADD COLUMN "+column+" integer NOT NULL DEFAULT 0"); err != nil {
				return err
			}
		}
		return forceFullRescan(ctx, tx)
	}, func(ctx context.Context, tx *sql.Tx) error {
		for _, column := range []string{"cue_source_rate", "cue_end_sample", "cue_start_sample"} {
			if _, err := tx.ExecContext(ctx, "ALTER TABLE media_file DROP COLUMN "+column); err != nil {
				return err
			}
		}
		return forceFullRescan(ctx, tx)
	})
}
