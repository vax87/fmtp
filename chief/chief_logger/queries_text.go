package chief_logger

import (
	"fmt"
	"strings"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/fmtp_logger"
)

const (
	onlineLogTableName  = "fmtp_online"
	storageLogTableName = "fmtp_storage"

	maxTextLen = 2000
)

// текст запроса кол-ва строк в таблице tableName.
func oraCheckLogCountQuery(maxOnlineCount int, maxStorageCount int) string {
	return fmt.Sprintf("BEGIN LOG_PROC_PKG.CHECK_LOG_COUNT(%d, %d); COMMIT; END;", maxOnlineCount, maxStorageCount)
}

// текст запроса удаления сообщений старше dateTimeString.
func oraCheckLogLivetimeQuery(dateTimeString string) string {
	return fmt.Sprintf("BEGIN LOG_PROC_PKG.CHECK_LOG_LIFETIME('%s'); COMMIT; END;", dateTimeString)
}

// текст запроса обновления состояния канала.
func oraUpdateChannelStateQuery(chState channel_state.ChannelState) string {
	var queryStr strings.Builder
	queryStr.WriteString("DECLARE ")
	queryStr.WriteString("chSt LOG_PROC_PKG.channel_state; ")
	queryStr.WriteString("BEGIN ")
	queryStr.WriteString(fmt.Sprintf("chSt.m_localname :='%s'; ", chState.LocalName))
	queryStr.WriteString(fmt.Sprintf("chSt.m_remotename := '%s'; ", chState.RemoteName))
	queryStr.WriteString(fmt.Sprintf("chSt.m_workingstate := '%s'; ", chState.DaemonState))
	queryStr.WriteString(fmt.Sprintf("chSt.m_fmtpstate := '%s'; ", chState.FmtpState))

	queryStr.WriteString("LOG_PROC_PKG.channel_state_changed(chSt); COMMIT; END;")
	return queryStr.String()
}

// текст запроса добавления сообщения журнала в БД.
func oraInsertLogQuery(logMessage fmtp_logger.LogMessage) string {
	queryText := "INSERT ALL "

	queryText += fmt.Sprintf(`INTO %s
		 (CntrlIP, DaemonID, LocalName, RemoteName, DataType, Source, Severity, FmtpType, Direction, DateTime, Text)
		 VALUES ('%s', '%d', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s') `,
		onlineLogTableName,
		logMessage.ControllerIP,
		logMessage.ChannelId,
		logMessage.ChannelLocName,
		logMessage.ChannelRemName,
		logMessage.DataType,
		logMessage.Source,
		logMessage.Severity,
		logMessage.FmtpType,
		logMessage.Direction,
		logMessage.DateTime,
		logMessage.Text)

	queryText += fmt.Sprintf(`INTO %s 
		(CntrlIP, DaemonID, LocalName, RemoteName, DataType, Source, Severity, FmtpType, Direction, DateTime, Text) 
		VALUES ('%s', '%d', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s') `,
		storageLogTableName,
		logMessage.ControllerIP,
		logMessage.ChannelId,
		logMessage.ChannelLocName,
		logMessage.ChannelRemName,
		logMessage.DataType,
		logMessage.Source,
		logMessage.Severity,
		logMessage.FmtpType,
		logMessage.Direction,
		logMessage.DateTime,
		logMessage.Text)

	queryText += " SELECT 1 FROM dual"

	return queryText
}

// текст запроса проверки подключени к БД.
func oraHeartbeatQuery() string {
	return "SELECT 'heartbeat' FROM dual"
}
