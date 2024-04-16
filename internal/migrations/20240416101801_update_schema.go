package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upUpdateSchema, downUpdateSchema)
}

func upUpdateSchema(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
		 ALTER TABLE parser RENAME COLUMN id_storis TO id_story;
		 ALTER TABLE parser ADD COLUMN created_at TIMESTAMP WITH TIME ZONE;
		 ALTER TABLE parser ADD COLUMN username VARCHAR;
 `)
	if err != nil {
		return err
	}
	return nil
}

func downUpdateSchema(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE parser RENAME COLUMN id_story TO id_storis;
		DROP COLUMN created_at;
		DROP COLUMN username;
	`)
	if err != nil {
		return err
	}
	return nil
}
