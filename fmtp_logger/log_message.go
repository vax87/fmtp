package fmtp_logger

import "time"

const (
	SeverityDebug   string = "Отладка"        // серьезность DEBUG.
	SeverityInfo    string = "Информация"     // серьезность INFO.
	SeverityWarning string = "Предупреждение" // серьезность WARNING.
	SeverityError   string = "Ошибка"         // серьезность ERROR.

	// направления сообщений
	DirectionIncoming  string = "Входящее"
	DirectionOutcoming string = "Исходящее"
	DirectionUnknown   string = "-"

	// источники сообщений сообщений
	SourceChannel    string = "FMTP канал"
	SourceController string = "Контроллер"

	/// текст идентификатора канала для "Контроллера"
	CntrlChannelIdentText string = "-"
	/// идентификатор канала для "Контроллера"
	NoChannelIdent int = -1

	NoChannelLocName string = "-"
	NoChannelRemName string = "-"

	NoneFmtpType string = "-"

	ChannelTypeNone string = "-"

	LogDebugColor   = "#e1e8f6"
	LogInfoColor    = "#eaf4e3"
	LogWarningColor = "#edecc3"
	LogErrorColor   = "#e7cfce"
	LogDefaultColor = "#EAECEE"

	LogTimeFormat = "2006-01-02 15:04:05.000"
)

// описание сообщенияя для журнала, получаемого по сети
type LogMessage struct {
	ControllerIP   string `json:"ControllerIP"` // IP адрес контроллера.
	Source         string `json:"Source"`       // название источника.
	ChannelId      int    `json:"DaemonID"`     // идентификатор канала.
	ChannelLocName string `json:"LocalName"`    // локальное имя канала.
	ChannelRemName string `json:"RemoteName"`   // удаленное имя канала.
	DataType       string `json:"DataType"`     // тип сообщений поверх FMTP.
	Severity       string `json:"Severity"`     // серьезность.
	FmtpType       string `json:"FmtpType"`     // тип FMTP пакета.
	Direction      string `json:"Direction"`    // направление сообщения.
	Text           string `json:"Text"`         // текст сообщения.
	DateTime       string `json:"DateTime"`     // дата и время сообщения.
}

// сообщение журнала с цветом
type LogMessageWithColor struct {
	LogMessage
	MsgColor string // цвет в таблице логов
}

// конструктор для использования к FMTP канале
func CreateMessage(severity string, packetType string,
	direction string, text string) LogMessage {
	var retValue LogMessage
	retValue.ControllerIP = ""
	retValue.Source = SourceChannel
	retValue.ChannelId = NoChannelIdent
	retValue.ChannelLocName = NoChannelLocName
	retValue.ChannelRemName = NoChannelRemName
	retValue.DataType = ChannelTypeNone
	retValue.Severity = severity
	retValue.FmtpType = packetType
	retValue.Direction = direction
	retValue.Text = text
	retValue.DateTime = time.Now().UTC().Format(LogTimeFormat) //"2006-01-02 15:04:05.333")
	return retValue
}

// конструктор для использования к FMTP канале
func CreateControllerMessage(severity string, text string) LogMessage {
	var retValue LogMessage
	retValue.ControllerIP = ""
	retValue.Source = SourceChannel
	retValue.ChannelId = NoChannelIdent
	retValue.ChannelLocName = NoChannelLocName
	retValue.ChannelRemName = NoChannelRemName
	retValue.DataType = ChannelTypeNone
	retValue.Severity = severity
	retValue.FmtpType = NoneFmtpType
	retValue.Direction = DirectionUnknown
	retValue.Text = text
	retValue.DateTime = time.Now().UTC().Format(LogTimeFormat) //"2006-01-02 15:04:05.333")
	return retValue
}


// LogChannelST сообщение с использование ST (Severity-Text)
func LogChannelST(severity string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  NoneFmtpType,
		Direction: DirectionUnknown,
		Text:      text,
		DateTime:  time.Now().UTC().Format(LogTimeFormat)}
}

// LogChannelSTDT сообщение с использование STDT (Severity-Type-Direction-Text)
func LogChannelSTDT(severity string, fmtpType string, direction string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  fmtpType,
		Direction: direction,
		Text:      text,
		DateTime:  time.Now().UTC().Format(LogTimeFormat)}
}

// LogCntrlSDT сообщение с использование SDT (Severity-DataType-Text)
func LogCntrlSDT(severity string, dtType string, text string) LogMessage {
	return LogMessage{
		ChannelId: NoChannelIdent,
		DataType:  dtType,
		FmtpType:  NoneFmtpType,
		Source:    SourceController,
		Severity:  severity,
		Direction: DirectionUnknown,
		Text:      text,
		DateTime:  time.Now().UTC().Format(LogTimeFormat)}
}
