package handlers

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouting(t *testing.T) {
	// инициализация хранилища, сервиса и сервера
	longURL := "https://translate.google.ru/?hl=ru&tab=wT&sl=ru&tl=en&text=%D0%A2%D0%B5%D1%81%D1%82%D0%B8%D1%80%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5&op=translate"
	initMap := map[string]string{"-testKey": longURL}
	cfg := config.New()
	cfg.DB.InitialData = initMap
	cfg.Server.BaseURL = "http://example.com"
	// os.Setenv("FILE_STORAGE_PATH", "C:/Users/annza/tempfile.storage")
	dataStorage := storage.NewMapDB(cfg.DB)
	service := shortener.New(dataStorage)
	handler := New(service, cfg.Server)
	srv := httptest.NewServer(handler.Router)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // клиент не следует по перенаправлениям
		},
	}
	defer srv.Close()

	type want struct {
		body       string
		statusCode int
		location   string
	}
	type req struct {
		method string
		url    string
		body   string
	}

	tt := []struct {
		name string
		req  req
		want want
	}{
		{
			name: "redirect 307",
			req:  req{method: "GET", url: "/-testKey"},
			want: want{statusCode: http.StatusTemporaryRedirect, location: longURL},
		},
		{
			name: "full url not found 400",
			req:  req{method: "GET", url: "/-noExistKey"},
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "create short link 201",
			req:  req{method: "POST", url: "/", body: longURL},
			// проверка body возможна только при фиксации rand.seed в тесте
			want: want{statusCode: http.StatusCreated, body: cfg.Server.BaseURL + "/23bS"},
		},
	}

	for _, d := range tt {
		t.Run(d.name, func(t *testing.T) {

			r, err := http.NewRequest(d.req.method, srv.URL+d.req.url, bytes.NewBufferString(d.req.body))
			require.NoError(t, err)
			// фиксируем сид чтобы получить ожидаемый сокращенный ключ
			rand.Seed(42)
			resp, err := client.Do(r)
			require.NoError(t, err)
			assert.Equal(t, d.want.statusCode, resp.StatusCode)

			assert.Equal(t, d.want.location, resp.Header.Get("Location"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, d.want.body, string(body))
		})
	}
}

func TestHandlers_HandlerApiShorten(t *testing.T) {
	cfg := config.New()
	cfg.Server.BaseURL = "http://example.com"
	dataStorage := storage.NewMapDB(cfg.DB)
	service := shortener.New(dataStorage)
	handler := New(service, cfg.Server)

	type want struct {
		body        string
		statusCode  int
		contentType string
	}
	type req struct {
		body        string
		contentType string
	}
	tests := []struct {
		name string
		req  req
		want want
	}{
		{
			name: "api json created link",
			req:  req{contentType: "application/json", body: `{"url":"https://translate.google.ru/?hl=ru&tab=wT&sl=ru&tl=en&text=%D1%82%D0%B5%D1%81%D1%82%20%20%20&op=translate"}`},
			want: want{statusCode: http.StatusCreated, contentType: "application/json", body: `{"result":"http://example.com/13bS"}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rand.Seed(42)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/shorten", bytes.NewBufferString(tt.req.body))
			r.Header.Set("Content-Type", "application/json")
			handler.HandlerAPIShorten(w, r)
			result := w.Result()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
			body, err := io.ReadAll(result.Body)
			result.Body.Close()
			require.NoError(t, err)
			assert.JSONEq(t, tt.want.body, string(body))
		})
	}
}
