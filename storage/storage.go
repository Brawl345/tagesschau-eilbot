package storage

import (
	"cmp"
	"embed"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

type DB struct {
	*sqlx.DB
	Subscribers SubscribersStorage
	SentNews    SentNewsStorage
}

func env(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func Connect() (*DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = env("MYSQL_USER")
	cfg.Passwd = env("MYSQL_PASSWORD")
	cfg.DBName = env("MYSQL_DB")
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	if socket := env("MYSQL_SOCKET"); socket != "" {
		cfg.Net = "unix"
		cfg.Addr = socket
	} else {
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(cmp.Or(env("MYSQL_HOST"), "127.0.0.1"), cmp.Or(env("MYSQL_PORT"), "3306"))
		cfg.TLSConfig = cmp.Or(env("MYSQL_TLS"), "false")
	}

	conn, err := sqlx.Connect("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(3 * time.Minute)
	conn.SetConnMaxIdleTime(time.Minute)

	return &DB{
		DB:          conn,
		Subscribers: &Subscribers{DB: conn},
		SentNews:    &SentNews{DB: conn},
	}, nil
}

func (db *DB) Migrate() (int, error) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	return migrate.Exec(db.DB.DB, "mysql", migrations, migrate.Up)
}
