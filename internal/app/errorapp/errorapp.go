// Package errorapp содержит кастомные ошибки, используемые в приложении.
package errorapp

import (
	"errors"
	"fmt"
)

// ErrorPageNotAvailable - возвращает ошибку, указывающую на то, что запрашиваемая страница больше не доступна.
var ErrorPageNotAvailable error = errors.New("запрашиваемая страница больше не доступна;")

// URLDuplicateError представляет ошибку, возникающую при попытке добавления дублирующегося URL.
type URLDuplicateError struct {
	Err error
	// ExistsKey - существующий ключ для URL
	ExistsKey string
	URL       string
}

// NewURLDuplicateError - создает и возвращает ссылку на ошибку URLDuplicateError
func NewURLDuplicateError(err error, key string, URL string) *URLDuplicateError {
	return &URLDuplicateError{Err: err, ExistsKey: key, URL: URL}
}

// Error - возвращает текстовое представление ошибки URLDuplicateError.
func (e *URLDuplicateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("для URL: %s уже существует короткий идентификатор %s; %v",
			e.URL, e.ExistsKey, e.Err.Error())
	}
	return fmt.Sprintf("для URL: %s уже существует короткий идентификатор %s", e.URL, e.ExistsKey)
}

// Unwrap - возвращает вложенную ошибку, если есть.
func (e *URLDuplicateError) Unwrap() error {
	return e.Err
}
