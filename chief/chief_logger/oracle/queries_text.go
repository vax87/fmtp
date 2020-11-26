package oracle

import (
	"fmt"
	"strings"

	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_logger/common"
)

const (
	onlineLogTableName  = "fmtp_online"
	storageLogTableName = "fmtp_storage"
	onlineLogViewName   = "fmtp_online_vw"
	storageLogViewName  = "fmtp_storage_vw"

	maxTextLen = 2000
)

// текст запроса кол-ва строк в таблице tableName.
func oraCheckLogCountQuery(maxOnlineCount int, maxStorageCount int) string {
	return fmt.Sprintf("BEGIN LOG_PROC_PKG.CHECK_LOG_COUNT(%d, %d); COMMIT; END;", maxOnlineCount, maxStorageCount)
}

// текст запроса удаления сообщений старше dateTimeString.
func oraCheckLogLifetimeQuery(dateTimeString string) string {
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

// текст начала запроса добавления сообщений в БД.
func oraInsertLogBeginQuery(queryText *string, logMessage common.LogMessage) {

	if *queryText == "" {
		*queryText += "INSERT ALL "
	}

	*queryText += fmt.Sprintf(`INTO %s
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

	*queryText += fmt.Sprintf(`INTO %s 
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
}

// текст окончания запроса добавления сообщений в БД.
func oraInsertLogEndQuery(queryText *string) {
	*queryText += " SELECT 1 FROM dual"
}
