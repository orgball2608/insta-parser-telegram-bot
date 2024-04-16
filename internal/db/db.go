package db

import (
	"database/sql"
	"fmt"
	"github.com/orgball2608/insta-parser-telegram-bot/pkg/config"
	"time"

	_ "github.com/lib/pq"
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

func (pg *Postgres) Check(id string, username string, createdAt time.Time) bool {
	var result bool
	if err := pg.db.QueryRow("SELECT result FROM parser WHERE id_story = $1", id).Scan(&result); err != nil {
		return false
	}

	if result {
		return true
	} else {
		_, err := pg.db.Exec("INSERT INTO parser(id_story, username, created_at, result) VALUES($1, $2, $3, true)", id, username, createdAt)
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
