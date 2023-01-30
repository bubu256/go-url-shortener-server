package postgres

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bubu256/go-url-shortener-server/config"
)

type PDStore struct {
	db               *sql.DB
	connectingString string
}

func New(cfg config.CfgDataBase) (*PDStore, error) {
	db, err := sql.Open("pgx", cfg.DataBaseDSN)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `CREATE TABLE IF NOT EXISTS urls(
		short_id CHAR(10) PRIMARY KEY NOT NULL,
		full_url TEXT,
		user_id CHAR(72) NOt NULL
	);`
	_, err = db.ExecContext(ctx, query)
	if err != nil {
		log.Println(err)
	}

	return &PDStore{connectingString: cfg.DataBaseDSN, db: db}, nil
}

func (p *PDStore) GetURL(key string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select full_url from urls where short_id = $1"
	row := p.db.QueryRowContext(ctx, query, key)
	if err := row.Err(); err != nil {
		log.Println(err)
		return "", false
	}
	fullURL := ""
	if row.Scan(&fullURL) != nil {
		return "", false
	}
	return fullURL, true
}

func (p *PDStore) GetAllURLs(userID string) map[string]string {
	result := make(map[string]string)
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select short_id, full_url from urls where user_id = $1"
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Println(err)
		return result
	}

	short := ""
	full := ""
	for rows.Next() {
		if rows.Scan(&short, &full) != nil {
			return result
		}
		// очень странно что тут появляются лишние проблемы у short
		// когда ответ однострочный такой проблемы нет
		result[strings.TrimSpace(short)] = full
	}
	if rows.Err() != nil {
		log.Println(err)
	}
	return result
}

func (p *PDStore) SetNewURL(key, URL, tokenID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "INSERT INTO urls (short_id, full_url, user_id) VALUES ($1, $2, $3)"
	_, err := p.db.ExecContext(ctx, query, key, URL, tokenID)
	return err
}

func (p *PDStore) GetLastID() (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select count(*) from urls"
	row := p.db.QueryRowContext(ctx, query)
	lastID := 0
	if err := row.Scan(&lastID); err != nil {
		return 0, false
	}
	return int64(lastID), true
}

func (p *PDStore) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	db, err := sql.Open("pgx", p.connectingString)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.PingContext(ctx)
}
