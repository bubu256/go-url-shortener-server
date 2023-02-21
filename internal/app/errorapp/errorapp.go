// пакет содержит кастомные ошибки приложения
package errorapp

import (
	"errors"
	"fmt"
)

var ErrorPageNotAvailable error = errors.New("запрашиваемая страница больше не доступна;")

type URLDuplicateError struct {
	Err       error
	ExistsKey string
	URL       string
}

func NewURLDuplicateError(err error, key string, URL string) *URLDuplicateError {
	return &URLDuplicateError{Err: err, ExistsKey: key, URL: URL}
}
func (e *URLDuplicateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("для URL: %s уже существует короткий идентификатор %s; %v",
			e.URL, e.ExistsKey, e.Err.Error())
	}
	return fmt.Sprintf("для URL: %s уже существует короткий идентификатор %s", e.URL, e.ExistsKey)
}

func (e *URLDuplicateError) Unwrap() error {
	return e.Err
}
