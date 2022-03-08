package storage

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type (
	SystemStorage interface {
		GetLastEntry() (string, error)
		SetLastEntry(lastEntry string) error
	}

	System struct {
		*sqlx.DB
	}
)

func (db *System) GetLastEntry() (string, error) {
	var lastEntry string
	err := db.Get(&lastEntry, "SELECT `value` FROM `system` WHERE `key` = 'last_entry'")
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
	}
	return lastEntry, err
}

func (db *System) SetLastEntry(lastEntry string) error {
	const query = "INSERT INTO `system` (`key`, `value`) VALUES ('last_entry', ?) ON DUPLICATE KEY UPDATE value = ?"
	_, err := db.Exec(query, lastEntry, lastEntry)
	return err
}
