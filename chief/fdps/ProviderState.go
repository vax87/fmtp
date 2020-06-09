package fdps

const (
	ProviderStateOk    = "ok"
	ProviderStateError = "error"
)

// состояние FDPS провайдера данных
type ProviderState struct {
	ProviderID           int      `json:"ProviderID"`    // идентификатор провайдера
	ProviderType         string   `json:"ProviderType"`  // тип провайдера (OLID | AODB)
	ProviderIPs          []string `json:"ProviderIPs"`   // список сетевых адресов провайдеров
	ProviderState        string   "ProviderState"        // состояние провайдера
	ProviderErrorMessage string   "ProviderErrorMessage" // текст ошибки
	ProviderURL          string   `json:"-"`             // URL web странички провайдера
}
