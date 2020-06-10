package common

import (
	"time"
)

// сообщение с использование ST (Severity-Text)
func LogChannelST(severity string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  NoneFmtpType,
		Direction: DirectionUnknown,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}

// сообщение с использование STDT (Severity-Type-Direction-Text)
func LogChannelSTDT(severity string, fmtpType string, direction string, text string) LogMessage {
	return LogMessage{
		Source:    SourceChannel,
		Severity:  severity,
		FmtpType:  fmtpType,
		Direction: direction,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}

// сообщение с использование ST (Severity-Text)
func LogCntrlST(severity string, text string) LogMessage {
	return LogMessage{
		ChannelId: NoChannelIdent,
		Source:    SourceController,
		Severity:  severity,
		FmtpType:  NoneFmtpType,
		Direction: DirectionUnknown,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}

// сообщение с использование STDT (Severity-Type-Direction-Text)
func LogCntrlSTDT(severity string, fmtpType string, direction string, text string) LogMessage {
	return LogMessage{
		ChannelId: NoChannelIdent,
		Source:    SourceController,
		Severity:  severity,
		FmtpType:  fmtpType,
		Direction: direction,
		Text:      text,
		DateTime:  time.Now().Format("2006-01-02 15:04:05.000")}
}
