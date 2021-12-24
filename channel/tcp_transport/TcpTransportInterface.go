package tcp_transport

import (
	"fdps/fmtp/fmtp"
	"fdps/fmtp/fmtp_logger"
)

// данные для отправки
type DataAndEvent struct {
	DataToSend     []byte         // данные для отправки
	EventAfterSend fmtp.FmtpEvent // событие, которое необходимо сгенерить после успешной отправки
}

// настройки клиентского/серверного TCP подключеия
type TcpTransportSettings struct {
	ServerAddr string // сетевой адрес (для клиента)
	ServerPort int    // сетевой порт (для клиента)

	ClientAddr string // сетевой адрес подключаемого лиента (для сервера)
	LocalPort  int    // сетевой порт для прослушивания (для сервера)
}

// интерфейс TCP транспорта
type TcpTransport interface {
	SettChan() chan TcpTransportSettings  // текущие настройки канала
	ReceivedChan() chan []byte            // канал для принятых данных
	SendChan() chan DataAndEvent          // канал для отправки данных по TCP
	EventChan() chan fmtp.FmtpEvent       // событие, генерируемое после отправки (кроме None)
	LogChan() chan fmtp_logger.LogMessage // канал для передачи сообщний для журнала
	ConnStateChan() chan bool             // канал для передачи успешности подключения по TCP
	ReconnectChan() chan struct{}         // канал для передачи сигнала о необходимости подключиться ксерверу (для TCP клиента)
	Work()
}
