package fmtp_logger

import (
	"fmt"
	"strconv"
	"time"

	"fdps/go_utils/logger"
)

// SetUserLogFormatForLjack() - свой формат вывода сообщений fmtp в файловый логгер
func SetUserLogFormatForLjack() {
	ljackFmtpFormatFunc := func(msgIface interface{}) ([]byte, bool) {
		var msgText string
		fmtpMsg, ok := msgIface.(LogMessage)
		if ok {
			msgText = time.Now().UTC().Format(LogTimeFormat)
			msgText += "\t " + fmtpMsg.ControllerIP + "\n"
			msgText += "\t Severity: " + fmtpMsg.Severity + "\n"
			msgText += "\t Source: " + fmtpMsg.Source + "\n"
			msgText += "\t ChannelId: " + strconv.Itoa(fmtpMsg.ChannelId) + "\n"
			msgText += "\t ChannelLocName: " + fmtpMsg.ChannelLocName + "\n"
			msgText += "\t ChannelRemName: " + fmtpMsg.ChannelRemName + "\n"
			msgText += "\t DataType: " + fmtpMsg.DataType + "\n"
			msgText += "\t FmtpType: " + fmtpMsg.FmtpType + "\n"
			msgText += "\t Direction: " + fmtpMsg.Direction + "\n"
			msgText += "\t Text: " + fmtpMsg.Text + "\n"
			msgText += "\n\n"
		}
		return []byte(msgText), ok
	}
	logger.SetLjackUserFormatFunc(ljackFmtpFormatFunc)
}

// SetUserLogFormatForWeb - свой формат вывода логов на web страницу
func SetUserLogFormatForWeb() {
	logger.WebLogSetTableUserFormat(`
		<div style="display: block;  height: 1000px; position: relative; overflow-x: auto;">
		<table width="100%" border="1" cellspacing="0" cellpadding="4" class="table table-bordered table-striped mb-0">
			<colgroup>
				<col span="1" style="width: 10%;">
				<col span="7" style="width: 5%;">
			</colgroup>
			<tr>
				<th>Дата, время</th>
				<th>Источник</th>
				<th>Серъезность</th>
				<th>Лок ATC</th>
				<th>Уд ATC</th>
				<th>Тип</th>
				<th>FMTP тип</th>
				<th>Направление</th>
				<th>Текст</th>
			</tr>
			{{with .Lr}}
				{{range .}}
					<tr align="center" bgcolor="{{.MsgColor}}">
						<td align="left"> {{.DateTime}}	</td>
						<td align="left"> {{.Source}} </td>
						<td align="left"> {{.Severity}}	</td>
						<td align="left"> {{.ChannelLocName}} </td>
						<td align="left"> {{.ChannelRemName}} </td>
						<td align="left"> {{.DataType}} </td>
						<td align="left"> {{.FmtpType}} </td>
						<td align="left"> {{.Direction}} </td>
						<td align="left"> {{.Text}} </td>
					</tr>
				{{end}}
			{{end}}
		</table>
	`)

	webUserFormatFunc := func(severity string, format string, a ...interface{}) interface{} {

		severityToColor := func(sev string) (msgColor string) {
			switch sev {
			case SeverityDebug:
				msgColor = logger.DebugColor
			case SeverityInfo:
				msgColor = logger.InfoColor
			case SeverityWarning:
				msgColor = logger.WarningColor
			case SeverityError:
				msgColor = logger.ErrorColor
			default:
				msgColor = logger.DefaultColor
			}
			return msgColor
		}

		var logValue interface{}
		if len(a) > 0 {
			fmtpLogMsg, ok := a[0].(LogMessage)
			if ok {
				logValue = LogMessageWithColor{
					LogMessage: fmtpLogMsg,
					MsgColor:   severityToColor(severity),
				}
			} else {
				logValue = LogMessageWithColor{
					LogMessage: CreateControllerMessage(severity, fmt.Sprintf(format, a...)),
					MsgColor:   severityToColor(severity),
				}
			}
		} else {
			logValue = LogMessageWithColor{
				LogMessage: CreateControllerMessage(severity, fmt.Sprintf(format, a...)),
				MsgColor:   severityToColor(severity),
			}
		}
		return logValue
	}
	logger.WebLogSetUserFormatFunc(webUserFormatFunc)
}
