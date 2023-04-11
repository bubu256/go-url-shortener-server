// Package schema предоставляет структуры, необходимые для пересылки данных между пакетами.
package schema

// APIShortenInput - структура, используемая для принятия данных в запросе
type APIShortenInput struct {
	URL string `json:"url"`
}

// APIShortenOutput - структура, используемая для отправки сокращенного URL в JSON.
type APIShortenOutput struct {
	Result string `json:"result"`
}

// APIUserURLs - массив структур, используемый для отправки всех URL пользователя в JSON.
type APIUserURLs []struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// APIShortenBatchInput - массив структур, используемый для приемки множества ссылок и их коротких идентификаторов.
type APIShortenBatchInput []struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// APIShortenBatchOutput - массив структур, используемый для возврата добавленных ссылок.
type APIShortenBatchOutput []struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
