package postgres

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/bubu256/go-url-shortener-server/pkg/helperfunc"
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
	log.Println("", cfg.DataBaseDSN)

	m, err := migrate.New(
		"file://db_migrate",
		cfg.DataBaseDSN,
	)
	if err != nil {
		log.Println("Не удалось подключиться к БД;", err)
		return nil, err
	}
	defer m.Close()
	if err := m.Up(); err == nil {
		log.Printf("Миграция применена к БД; %v", m)
	}

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

// func (p *PDStore) DeleteBatch(batchShortKeys []string, token string) error {
func (p *PDStore) DeleteBatch(inputChs []chan []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qwr := `UPDATE urls 
	SET full_url = short_id||'_deleted='||full_url,
	available = FALSE 
	WHERE user_id = $1 and short_id = $2 and available = TRUE
	`
	stmt, err := tx.PrepareContext(ctx, qwr)
	if err != nil {
		return err
	}

	var errOut error
	for keyUser := range helperfunc.FanInSliceString(inputChs...) {
		_, err = stmt.ExecContext(ctx, keyUser[1], keyUser[0])
		if err != nil {
			// если возникла ошибка мы все равно продолжаем вычитывать канал,
			// чтобы он смог безопасно закрыться
			log.Println(err)
			errOut = err
		}
	}
	// Проверяем были ли ошибки
	if errOut != nil {
		return err
	}

	return tx.Commit()
}

func (p *PDStore) GetURL(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select full_url, available from urls where short_id = $1"
	row := p.db.QueryRowContext(ctx, query, key)
	if err := row.Err(); err != nil {
		log.Println(err)
		return "", err
	}
	fullURL := ""
	available := false
	err := row.Scan(&fullURL, &available)
	if err != nil {
		return "", err
	}
	if !available {
		return "", errorapp.ErrorPageNotAvailable
	}
	return fullURL, nil
}

func (p *PDStore) GetAllURLs(userID string) map[string]string {
	result := make(map[string]string)
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select short_id, full_url, available from urls where user_id = $1"
	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Println(err)
		return result
	}

	short := ""
	full := ""
	available := false
	for rows.Next() {
		if rows.Scan(&short, &full, &available) != nil {
			return result
		}
		if !available {
			continue
		}
		// до сих пор не понимаю почему тут появляются лишние проблемы у short
		// когда ответ однострочный такой проблемы нет
		result[strings.TrimSpace(short)] = full
	}
	if rows.Err() != nil {
		log.Println(err)
	}
	return result
}

func (p *PDStore) SetNewURL(key, URL, tokenID string, available bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "INSERT INTO urls (short_id, full_url, user_id, available) VALUES ($1, $2, $3, $4)"
	_, err := p.db.ExecContext(ctx, query, key, URL, tokenID, available)
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
