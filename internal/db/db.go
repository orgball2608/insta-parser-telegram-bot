package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/orgball2608/insta-parser-telegram-bot/internal/config"
	_ "github.com/orgball2608/insta-parser-telegram-bot/internal/migrations"
	"github.com/pressly/goose/v3"
)

type Postgres struct {
	db *sql.DB
}

func NewConnect(cfg *config.Config) (*Postgres, error) {
	connect, err := sql.Open("postgres", fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d sslmode=%s ", cfg.Postgres.Name, cfg.Postgres.User, cfg.Postgres.Pass, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.SslMode))
	if err != nil {
		return nil, err
	}

	if err = connect.Ping(); err != nil {
		return nil, err
	}

	return &Postgres{db: connect}, nil
}

func (pg *Postgres) Check(id string) bool {
	var result bool
	pg.db.QueryRow("SELECT result FROM parser WHERE id_storis = $1", id).Scan(&result)

	if result {
		return true
	} else {
		_, err := pg.db.Exec("INSERT INTO parser(id_storis, result) VALUES($1, true)", id)
		if err != nil {
			return false
		}
	}

	return false
}

func (pg *Postgres) MigrationInit() error {
	err := goose.Up(pg.db, ".")
	if err != nil {
		return err
	}

	return nil
}
