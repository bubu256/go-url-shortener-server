// данный пакет содержит структуры
// необходимые для пересылки данных между другими пакетами
package data

// структура для принятия данных в запросе из json
type InputData struct {
	URL string `json:"url"`
}

// структура для отправки сокращенного url в json
type ApiShorten struct {
	Result string `json:"result"`
}

// структура для отправки всех url пользователя в json
type ApiUserURLs []struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// структура для приемки множества ссылок и их коротких идентификаторов
type ApiShortenBatch []struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}
