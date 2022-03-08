package storage

import (
	"errors"
	"github.com/jmoiron/sqlx"
)

type (
	SubscribersStorage interface {
		Create(chatId int64) error
		Delete(chatId int64) error
		Exists(chatId int64) (bool, error)
		GetAll() ([]int64, error)
	}

	Subscribers struct {
		*sqlx.DB
	}
)

func (db *Subscribers) Create(chatId int64) error {
	const query = `INSERT INTO subscribers (id) VALUES (?)`
	_, err := db.Exec(query, chatId)
	return err
}

func (db *Subscribers) Delete(chatId int64) error {
	const query = `DELETE FROM subscribers WHERE id = ?`
	res, err := db.Exec(query, chatId)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if rows == 0 {
		return errors.New("subscriber not found")
	}
	return err
}

func (db *Subscribers) Exists(chatId int64) (bool, error) {
	const query = `SELECT 1 FROM subscribers
WHERE subscribers.id = ?`

	var exists bool
	err := db.Get(&exists, query, chatId)
	return exists, err
}

func (db *Subscribers) GetAll() ([]int64, error) {
	const query = `SELECT id FROM subscribers`

	var ids []int64
	err := db.Select(&ids, query)
	return ids, err
}
