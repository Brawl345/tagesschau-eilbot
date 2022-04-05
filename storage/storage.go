package storage

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

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

func Connect() (*DB, error) {
	host := strings.TrimSpace(os.Getenv("MYSQL_HOST"))
	port := strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	user := strings.TrimSpace(os.Getenv("MYSQL_USER"))
	password := strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	db := strings.TrimSpace(os.Getenv("MYSQL_DB"))

	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		db,
	)

	conn, err := sqlx.Connect("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(100)
	conn.SetMaxOpenConns(100)
	conn.SetConnMaxIdleTime(3 * time.Minute)

	return &DB{
		DB:          conn,
		Subscribers: &Subscribers{DB: conn},
		System:      &System{DB: conn},
	}, nil
}

func (db *DB) Migrate() (int, error) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	return migrate.Exec(db.DB.DB, "mysql", migrations, migrate.Up)
}
