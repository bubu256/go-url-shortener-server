package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortener(t *testing.T) {
	type want struct {
		code              int
		HeaderContentType string
		response          string
		HeaderLocation    string
	}

	tests := []struct {
		name    string
		request *http.Request
		mapURL  map[string]string
		want    want
	}{
		{
			name:    "got short link",
			request: httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://practicum.yandex.ru/catalog/")),
			want:    want{code: 201, response: "http://example.com/1"},
		},
		{
			name:    "redirect",
			request: httptest.NewRequest(http.MethodGet, "/1", nil),
			mapURL:  map[string]string{"1": "https://practicum.yandex.ru/catalog/"},
			want:    want{code: 307, HeaderLocation: "https://practicum.yandex.ru/catalog/"},
		},
		{
			name:    "code 400",
			request: httptest.NewRequest(http.MethodGet, "/2", nil),
			mapURL:  map[string]string{"1": "https://practicum.yandex.ru/catalog/"},
			want:    want{code: 400},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mapURL != nil {
				// затеняем baseURL для теста
				baseURL := tt.mapURL
				_ = baseURL
			}
			w := httptest.NewRecorder()
			http.HandlerFunc(Shortener).ServeHTTP(w, tt.request)
			res := w.Result()
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.HeaderContentType, res.Header.Get("Content-Type"))
			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.NoError(t, err)
			assert.Equal(t, tt.want.response, string(body))
		})
	}
}
