package storage

import (
	"embed"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

type DB struct {
	*sqlx.DB
	Subscribers SubscribersStorage
	System      SystemStorage
}

func Open(url string) (*DB, error) {
	db, err := sqlx.Open("mysql", url)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(100)
	db.SetMaxOpenConns(100)

	return &DB{
		DB:          db,
		Subscribers: &Subscribers{db},
		System:      &System{db},
	}, nil
}

func (db *DB) Migrate() (int, error) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	return migrate.Exec(db.DB.DB, "mysql", migrations, migrate.Up)
}
