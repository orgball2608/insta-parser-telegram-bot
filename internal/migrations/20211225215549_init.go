package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upInit, downInit)
}

func upInit(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE parser (id SERIAL, id_storis VARCHAR, result BOOLEAN);
	`)
	if err != nil {
		return err
	}
	return nil
}

func downInit(tx *sql.Tx) error {
	_, err := tx.Exec(`
	DROP TABLE parser;
	`)
	if err != nil {
		return err
	}
	return nil
}
