package fdps

const (
	// AODBProvider значение типа провайдера
	AODBProvider = "AODB"
	// OLDIProvider значение типа провайдера
	OLDIProvider = "OLDI"
)

// настройки провайдера данных
type ProviderSettings struct {
	ID               int      `json:"ProviderID"`     // идентификатор провайдера
	IPAddresses      []string `json:"ProviderIPs"`    // список IP адресов провайдера
	Status           string   `json:"ProviderStatus"` // статус работы провайдера (primary/secondary) - не используется
	DataType         string   `json:"ProviderType"`   // тип данных поверх FMTP ("AODB" | "OLDI")
	ProviderEncoding string   // кодировка сообщений при общении с провайдером OLDI ("Windows-1251" | "UTF-8")
	LocalPort        int      // сетевой порт (заполняется из общей структуры настроек)
}

// настройки провайдера данных AODB
// type AodbProviderSettings struct {
// 	Id          int      // идентификатор провайдера
// 	IpAddresses []string // список IP адресов провайдера
// 	LocalPort   int      // сетевой порт (заполняется из общей структуры настроек)
// }

// настройки провайдера данных OLDI
// type OldiProviderSettings struct {
// 	Id          int      // идентификатор провайдера
// 	IpAddresses []string // список IP адресов провайдера
// 	LocalPort   int      // сетевой порт (заполняется из общей структуры настроек)
// }
