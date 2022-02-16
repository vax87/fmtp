package channel_settings

import (
	"errors"
	"strconv"

	"fmtp/fmtp"
)

const (
	startText string = "running"
	stopText  string = "stopped"

	yesText string = "yes"
	noText  string = "no"

	TcpServerText  string = "server"
	TcpClientText  string = "client"
	TcpUnknownText string = "unknown"

	Encode1251 string = "Windows-1251"
	EncodeUtf  string = "UTF-8"
)

// ChannelSettings настройки контроллера записи логов в файл
type ChannelSettings struct {
	Id               int            `json:"DaemonID"`         // идентифкаторо демона (получен от конфигуратора). *Не менять на  ChannelId(из конфигуратора будет приходить 0)
	Version          string         `json:"Version"`          // название версии fmtp демона.
	DataType         string         `json:"DataType"`         // тип сообщений поверх FMTP.
	NetRole          string         `json:"NetRole"`          // тип TCP подключения.
	LocalName        string         `json:"LocalName"`        // локальное имя.
	LocalATC         string         `json:"LocalATC"`         // локальный АТС.
	RemoteName       string         `json:"RemoteName"`       // удаленное имя.
	RemoteATC        string         `json:"RemoteATC"`        // удаленный АТС.
	IntervalTs       int            `json:"Ts"`               // таймаут таймера Ts.
	IntervalTr       int            `json:"Tr"`               // таймаут таймера Tr.
	IntervalTi       int            `json:"Ti"`               // таймаут таймера Ti.
	ReconnectTimeout int            `json:"ReconnectTimeout"` // таймаут подключения (для клиента).
	FmtpInitStateStr string         `json:"FmtpInitState"`    // текст 'data_ready' | 'disabled'
	FmtpInitState    fmtp.FmtpState // начальное состояние.
	RemoteAddress    string         `json:"RemoteAddress"` // удаленный адрес.
	RemotePort       int            `json:"RemotePort"`    // удаленный порт (для клиента).
	LocalPort        int            `json:"LocalPort"`     // локальный порт	(для сервера).
	DataEncoding     string         `json:"DataEncoding"`  // кодировка сообщений.
	LogDebug         bool           `json:"DebugLog"`      // с отладочными сообщениями.
	IsWorking        bool           `json:"State"`         // работоспособность.
	URLAddress       string         `json:"URLAddress"`    // IP адрес для доступа к web страничке
	URLPath          string         `json:"URLPath"`       // путь для доступа к web страничке
	URLPort          int            `json:"URLPort"`       // порт для доступа к web страничке
}

// ToLogMessage строка для вывода в лог.
func (chSett *ChannelSettings) ToLogMessage() string {
	var retValue string
	retValue += "Версия: " + chSett.Version + ","
	retValue += "Тип канала: " + chSett.DataType + ","
	retValue += "Тип TCP: " + chSett.NetRole + ","
	retValue += "Локальное имя: " + chSett.LocalName + " ,"
	retValue += "Локальный ATC: " + chSett.LocalATC + " ,"
	retValue += "Удаленное имя: " + chSett.RemoteName + " ,"
	retValue += "Удаленный АТС: " + chSett.RemoteATC + " ,"
	retValue += "Интервал Ts: " + strconv.Itoa(chSett.IntervalTs) + " ,"
	retValue += "Интервал Tr: " + strconv.Itoa(chSett.IntervalTr) + " ,"
	retValue += "Интервал Ti: " + strconv.Itoa(chSett.IntervalTi) + " ,"
	retValue += "Начальное состояние: " + chSett.FmtpInitState.ToString() + " ,"
	retValue += "Удаленный адрес: " + chSett.RemoteAddress + " "
	if chSett.NetRole == TcpClientText {
		retValue += "Удаленный порт: " + strconv.Itoa(chSett.RemotePort) + " "
	}
	if chSett.NetRole == TcpClientText {
		retValue += "Интервал подключения: " + strconv.Itoa(chSett.ReconnectTimeout) + " ,"
		retValue += "Локальный порт: " + strconv.Itoa(chSett.LocalPort) + " "
	}
	retValue += "Кодировка: " + chSett.DataEncoding + " "
	retValue += "Отладка: "
	if chSett.LogDebug {
		retValue += " да. "
	} else {
		retValue += " нет. "
	}
	retValue += "Работоспособность: "
	if chSett.IsWorking {
		retValue += " да. "
	} else {
		retValue += " нет. "
	}
	return retValue
}

// CheckSettings проверка настроек после unmarshall
func (chSett *ChannelSettings) CheckSettings() error {
	if chSett.DataType == "" {
		return errors.New("Не задан тип канала.")
	}
	if chSett.NetRole != TcpServerText &&
		chSett.NetRole != TcpClientText {
		return errors.New("Не задан тип TCP соединения.")
	}
	if chSett.LocalName == "" {
		return errors.New("Не задано локальное имя канала.")
	}
	if chSett.LocalATC == "" {
		return errors.New("Не задано названия локального ATC имя канала.")
	}
	if chSett.RemoteName == "" {
		return errors.New("Не задано удаленное имя канала.")
	}
	if chSett.RemoteATC == "" {
		return errors.New("Не задано названия удаленного ATC имя канала.")
	}
	if chSett.IntervalTs >= chSett.IntervalTr {
		return errors.New("Значение Ts должно быть меньше значения Tr")
	}
	if chSett.NetRole == TcpServerText &&
		(chSett.LocalPort <= 2000 || chSett.LocalPort >= 65535) {
		return errors.New("Некорректное значение локального порта.")
	}
	if chSett.NetRole == TcpClientText && chSett.RemoteAddress == "" {
		return errors.New("Не указан адрес удаленного АРМ.")
	}
	if chSett.NetRole == TcpClientText &&
		(chSett.RemotePort <= 2000 || chSett.RemotePort >= 65535) {
		return errors.New("Некорректное значение порта удаленнного АРМ.")
	}
	if chSett.DataEncoding == "" {
		return errors.New("Не задана кодировка сообщений.")
	}

	chSett.FmtpInitState.FromString(chSett.FmtpInitStateStr)
	return nil
}

// ChannelSettingsWithPort настройки каналов, плюс порт для взяимодействия с каналами
type ChannelSettingsWithPort struct {
	ChSettings []ChannelSettings
	ChPort     int
}
