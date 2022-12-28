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
	longURL := "https://translate.google.ru/?hl=ru&tab=wT&sl=ru&tl=en&text=%D0%A2%D0%B5%D1%81%D1%82%D0%B8%D1%80%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5&op=translate"
	initMap := map[string]string{"-testKey": longURL}
	cfg := config.New()
	cfg.DB.InitialData = initMap

	dataStorage := storage.NewMapDBMutex(cfg.DB)
	service := shortener.New(dataStorage)
	handler := New(service)
	srv := httptest.NewServer(handler.Router)
	// клиент не следует по перенаправлениям
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
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
			// проверка body возможна токи при фиксации rand.seed в тесте
			want: want{statusCode: http.StatusCreated, body: srv.URL + "/q0V3bS"},
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
