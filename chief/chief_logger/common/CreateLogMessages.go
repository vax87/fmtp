package common

import (
	"time"
)

// LogChannelST сообщение с использование ST (Severity-Text)
func LogChannelST(severity string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  NoneFmtpType,
		Direction: DirectionUnknown,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}

// LogChannelSTDT сообщение с использование STDT (Severity-Type-Direction-Text)
func LogChannelSTDT(severity string, fmtpType string, direction string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  fmtpType,
		Direction: direction,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
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
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}
