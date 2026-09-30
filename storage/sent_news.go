package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

const sentNewsKeep = 100

type (
	SentNewsStorage interface {
		IsEmpty(ctx context.Context) (bool, error)
		MarkSent(ctx context.Context, id string) (bool, error)
		Prune(ctx context.Context) error
	}

	SentNews struct {
		*sqlx.DB
	}
)

func (db *SentNews) IsEmpty(ctx context.Context) (bool, error) {
	var exists bool
	err := db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM `sent_news`)")
	return !exists, err
}

// MarkSent reports whether the ID was newly recorded.
func (db *SentNews) MarkSent(ctx context.Context, id string) (bool, error) {
	res, err := db.ExecContext(ctx, "INSERT IGNORE INTO `sent_news` (`id`) VALUES (?)", id)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows == 1, err
}

func (db *SentNews) Prune(ctx context.Context) error {
	var cutoff time.Time
	err := db.GetContext(ctx, &cutoff,
		"SELECT `created_at` FROM `sent_news` ORDER BY `created_at` DESC LIMIT 1 OFFSET ?", sentNewsKeep-1)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "DELETE FROM `sent_news` WHERE `created_at` < ?", cutoff)
	return err
}
