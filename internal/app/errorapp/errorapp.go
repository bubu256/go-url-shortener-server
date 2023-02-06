// пакет содержит кастомные ошибки приложения
package errorapp

import (
	"errors"
	"fmt"
)

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

func (e *URLDuplicateError) Is(target error) bool {
	if target == nil {
		return false
	}
	_, ok := target.(*URLDuplicateError)
	if ok {
		return true
	}
	// return false
	// как правильно? Нужно самому распаковывать ошибки и проверять глубже как тут или это делает сам errors.Is(err, target)
	unwrapErr := errors.Unwrap(target)
	if unwrapErr != nil {
		return e.Is(unwrapErr)
	}
	return false

}

func (e *URLDuplicateError) As(target interface{}) bool {
	_, ok := target.(*URLDuplicateError)
	return ok
}
