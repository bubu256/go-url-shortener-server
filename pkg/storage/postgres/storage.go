package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
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
	// "host=localhost port=5432 user=postgres password=myPassword dbname=postgres sslmode=disable"
	log.Println(cfg.DataBaseDSN)
	dsn := strings.Split(cfg.DataBaseDSN, " ")

	m, err := migrate.New(
		"file://db_migrate",
		// "postgres://user:password@host:port/db_name?sslmode=disable")
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dsn[2][5:], dsn[3][9:], dsn[0][5:], dsn[1][5:], dsn[4][7:], dsn[5][8:]),
	)
	if err != nil {
		log.Println("Не удалось подключиться к БД")
		log.Fatal(err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		log.Printf("Миграции не применены; %v", err)
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	// defer cancel()
	// query := `CREATE TABLE IF NOT EXISTS urls(
	// 	short_id CHAR(50) PRIMARY KEY NOT NULL,
	// 	full_url TEXT UNIQUE,
	// 	user_id CHAR(72) NOt NULL
	// );`
	// _, err = db.ExecContext(ctx, query)
	// if err != nil {
	// 	log.Println(err)
	// }

	return &PDStore{connectingString: cfg.DataBaseDSN, db: db}, nil
}

func (p *PDStore) SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error) {
	result := make([]string, 0, len(batch))
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stmnt, err := tx.Prepare("INSERT INTO urls (short_id, full_url, user_id) VALUES ($1, $2, $3)")
	if err != nil {
		return nil, err
	}
	find, err := tx.Prepare("select 1 from urls where short_id = $1")
	if err != nil {
		return nil, err
	}
	defer stmnt.Close()
	for _, elem := range batch {
		var isFinded bool
		find.QueryRow(elem.CorrelationID).Scan(&isFinded)
		if isFinded {
			continue
		}
		_, err := stmnt.Exec(elem.CorrelationID, elem.OriginalURL, token)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		result = append(result, elem.CorrelationID)
	}
	tx.Commit()
	return result, nil
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
	if err != nil && strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
		query := "select short_id from urls where full_url = $1 "
		var key string
		err := p.db.QueryRowContext(ctx, query, URL).Scan(&key)
		if err != nil {
			return err
		}
		return errorapp.NewURLDuplicateError(err, strings.TrimSpace(key), URL)
	}
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
