package storage

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type (
	SubscribersStorage interface {
		Create(ctx context.Context, chatId int64) (bool, error)
		Delete(ctx context.Context, chatId int64) (bool, error)
		GetAll(ctx context.Context) ([]int64, error)
	}

	Subscribers struct {
		*sqlx.DB
	}
)

// Create reports whether the subscriber was newly added.
func (db *Subscribers) Create(ctx context.Context, chatId int64) (bool, error) {
	return db.affected(ctx, "INSERT IGNORE INTO `subscribers` (`id`) VALUES (?)", chatId)
}

// Delete reports whether a subscriber was removed.
func (db *Subscribers) Delete(ctx context.Context, chatId int64) (bool, error) {
	return db.affected(ctx, "DELETE FROM `subscribers` WHERE `id` = ?", chatId)
}

func (db *Subscribers) GetAll(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := db.SelectContext(ctx, &ids, "SELECT `id` FROM `subscribers`")
	return ids, err
}

func (db *Subscribers) affected(ctx context.Context, query string, args ...any) (bool, error) {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows > 0, err
}
