package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func TestHandlerURLtoShort(t *testing.T) {
	tt := []struct {
		name     string
		url      string
		body     string
		wantCode int
		wantBody string
	}{
		{
			name:     "got valid shortlink",
			url:      "/",
			body:     "https://translate.google.ru/?hl=ru&tab=wT&sl=ru&tl=en&text=%D0%AD%D1%82%D0%BE%20%D1%82%D0%B5%D1%81%D1%82%D0%B8%D1%80%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5%20%D0%BF%D0%BE%D0%BB%D1%83%D1%87%D0%B5%D0%BD%D0%B8%D1%8F%20%D0%BA%D0%BE%D1%80%D0%BE%D1%82%D0%BA%D0%BE%D0%B9%20%D1%81%D1%81%D1%8B%D0%BB%D0%BA%D0%B8&op=translate",
			wantCode: 201,
		},
	}
	for _, d := range tt {
		t.Run(d.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", d.url, bytes.NewBufferString(d.body))
			http.HandlerFunc(HandlerURLtoShort).ServeHTTP(w, r)
			res := w.Result()
			assert.Equal(t, d.wantCode, res.StatusCode)
			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			require.NoError(t, err)
			isValidUrl := IsUrl(string(body))
			assert.True(t, isValidUrl)
		})
	}
}

func TestHandlerShortToFullURL(t *testing.T) {
	longURL := "https://translate.google.ru/?hl=ru&tab=wT&sl=ru&tl=en&text=%D0%AD%D1%82%D0%BE%20%D1%82%D0%B5%D1%81%D1%82%D0%B8%D1%80%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5%20%D0%BF%D0%BE%D0%BB%D1%83%D1%87%D0%B5%D0%BD%D0%B8%D1%8F%20%D0%BA%D0%BE%D1%80%D0%BE%D1%82%D0%BA%D0%BE%D0%B9%20%D1%81%D1%81%D1%8B%D0%BB%D0%BA%D0%B8&op=translate"
	baseURL = map[string]string{"1": longURL}

	tt := []struct {
		name         string
		url          string
		wantCode     int
		wantLocation string
	}{
		{
			name:         "redirect to base url",
			url:          "/1",
			wantCode:     307,
			wantLocation: longURL,
		},
		{
			name:     "status code 400",
			url:      "/0",
			wantCode: 400,
		},
	}
	for _, d := range tt {
		t.Run(d.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", d.url, nil)
			http.HandlerFunc(HandlerShortToFullURL).ServeHTTP(w, r)
			res := w.Result()
			assert.Equal(t, d.wantCode, res.StatusCode)
			assert.Equal(t, d.wantLocation, res.Header.Get("Location"))
		})
	}
}
