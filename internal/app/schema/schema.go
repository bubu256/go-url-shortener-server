// данный пакет содержит структуры
// необходимые для пересылки данных между другими пакетами
package schema

// структура для принятия данных в запросе из json
type APIShortenInput struct {
	URL string `json:"url"`
}

// структура для отправки сокращенного url в json
type APIShortenOutput struct {
	Result string `json:"result"`
}

// структура для отправки всех url пользователя в json
type APIUserURLs []struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// структура для приемки множества ссылок и их коротких идентификаторов
type APIShortenBatchInput []struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// структура для возврата добавленных ссылок
type APIShortenBatchOutput []struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
