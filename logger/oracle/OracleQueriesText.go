package oracle

import (
	"strconv"
	//	"strings"

	"fdps/fmtp/logger/common"
)

// создать таблицу с логами.
func oraCreateLogTableQuery(tableName string) string {
	return string("CREATE TABLE " + tableName +
		" ( LogId NUMBER NOT NULL, " +
		"CntrlIP NVARCHAR2(15), " +
		"DaemonID NUMBER, " +
		"LocalName NVARCHAR2(10), " +
		"RemoteName NVARCHAR2(10), " +
		"DataType NVARCHAR2(15), " +
		"Source NVARCHAR2(15), " +
		"Severity NVARCHAR2(15) NOT NULL, " +
		"FmtpType NVARCHAR2(15), " +
		"Direction NVARCHAR2(15), " +
		"DateTime NVARCHAR2(30) NOT NULL, " +
		"Text NVARCHAR2(500) NOT NULL)")
}

// создать первичный ключ в таблице логов.
func oraCreatePrimaryKeyQuery(tableName string) string {
	return string("ALTER TABLE " + tableName + " ADD CONSTRAINT " + tableName + "_PK PRIMARY KEY(LOGID)")
}

// создать представление с логами.
func oraCreateLogViewQuery(viewName string, tableName string) string {
	return string("CREATE OR REPLACE VIEW " + viewName + " AS SELECT " +
		"CntrlIP as cntrlip, " +
		"DaemonID as daemonid, " +
		"DataType as datatype, " +
		"LocalName as localname, " +
		"RemoteName as remotename, " +
		"Direction as direction, " +
		"FmtpType as fmtptype, " +
		"Severity as severity, " +
		"Source as source, " +
		"DateTime as datetime, " +
		"Text as text " +
		"from " + tableName)
}

// создать последовательность для автоинкремента.
func oraCreateSequenceQuery(tableName string) string {
	return string("CREATE SEQUENCE " + tableName + "_ID_SEQ START WITH 1 NOCACHE ORDER")
}

// создать триггер для автоинкремента.
func oraCreateTriggerQuery(tableName string) string {
	return string("CREATE OR REPLACE TRIGGER " + tableName + "_ID_TRG BEFORE " +
		"INSERT ON " + tableName + " " +
		"FOR EACH ROW " +
		"WHEN(NEW.LOGID IS NULL) " +
		"BEGIN " +
		":NEW.LOGID := " + tableName + "_ID_SEQ.NEXTVAL;" +
		"END;")
}

// текст запроса кол-ва строк в таблице tableName.
func oraCheckLogCountQuery(maxOnlineCount int, maxStorageCount int) string {
	return string("BEGIN LOG_PROC_PKG.CHECK_LOG_COUNT(" +
		strconv.Itoa(maxOnlineCount) + ", " + strconv.Itoa(maxStorageCount) + "); COMMIT; END;")
}

// текст запроса удаления сообщений старше dateTimeString.
func oraCheckLogLifetimeQuery(dateTimeString string) string {
	return string("BEGIN LOG_PROC_PKG.CHECK_LOG_LIFETIME('" + dateTimeString + "'); COMMIT; END;")
}

// текст начала запроса добавления сообщений в БД.
func oraInsertLogBeginQuery(queryText *string, logMessage common.LogMessage) {

	if *queryText == "" {
		*queryText += string("INSERT ALL ")
	}

	*queryText += string("INTO " + onlineLogTableName +
		" (CntrlIP, DaemonID, LocalName, RemoteName, DataType, Source, Severity, FmtpType, Direction, DateTime, Text) " +
		" VALUES ('" + logMessage.ControllerIP + "', '" +
		strconv.Itoa(logMessage.ChannelId) + "', '" +
		logMessage.ChannelLocName + "', '" +
		logMessage.ChannelRemName + "', '" +
		logMessage.DataType + "', '" +
		logMessage.Source + "', '" +
		logMessage.Severity + "', '" +
		logMessage.FmtpType + "', '" +
		logMessage.Direction + "', '" +
		//currentDateTime + "', '" +
		logMessage.DateTime + "', '" +
		logMessage.Text + "') ")

	*queryText += string("INTO " + storageLogTableName +
		" (CntrlIP, DaemonID, LocalName, RemoteName, DataType, Source, Severity, FmtpType, Direction, DateTime, Text) " +
		" VALUES ('" + logMessage.ControllerIP + "', '" +
		strconv.Itoa(logMessage.ChannelId) + "', '" +
		logMessage.ChannelLocName + "', '" +
		logMessage.ChannelRemName + "', '" +
		logMessage.DataType + "', '" +
		logMessage.Source + "', '" +
		logMessage.Severity + "', '" +
		logMessage.FmtpType + "', '" +
		logMessage.Direction + "', '" +
		//currentDateTime + "', '" +
		logMessage.DateTime + "', '" +
		logMessage.Text + "') ")
}

// текст окончания запроса добавления сообщений в БД.
func oraInsertLogEndQuery(queryText *string) {
	*queryText += string(" SELECT 1 FROM dual")
}

// текст заголовка пакета для удаления старых и лишних логов
func oraPackageQuery() string {
	return string(
		"CREATE OR REPLACE PACKAGE LOG_PROC_PKG AS\n" +
			"	--проверка кол-ва логов\n" +
			"   PROCEDURE CHECK_LOG_COUNT( MAX_ONLINE_LOG_COUNT IN NUMBER,	MAX_STORAGE_LOG_COUNT IN NUMBER	);\n" +
			"	--проверка времени хранения логов\n" +
			"   PROCEDURE CHECK_LOG_LIFETIME( OLDEST_STORAGE_LOG_DATE IN FMTP_STORAGE.DATETIME%TYPE);\n" +
			"END;")
}

// текст тела пакета для удаления старых и лишних логов
func oraPackageBodyQuery() string {
	return string(
		"CREATE OR REPLACE PACKAGE BODY LOG_PROC_PKG AS\n" +
			"	--проверка кол-ва логов\n" +
			"	PROCEDURE CHECK_LOG_COUNT( MAX_ONLINE_LOG_COUNT IN NUMBER,	MAX_STORAGE_LOG_COUNT IN NUMBER	)\n" +
			"	AS\n" +
			"		CUR_ONLINE_LOG_COUNT   NUMBER;\n" +
			"		CUR_STORAGE_LOG_COUNT   NUMBER;\n" +
			"       MIN_ONLINE_LOG_ID   FMTP_ONLINE.DATETIME%TYPE;\n" +
			"       MIN_STORAGE_LOG_ID   FMTP_STORAGE.DATETIME%TYPE;\n" +
			"	BEGIN\n" +
			"		SELECT COUNT(*) INTO CUR_ONLINE_LOG_COUNT FROM FMTP_ONLINE;\n" +
			"		IF CUR_ONLINE_LOG_COUNT > MAX_ONLINE_LOG_COUNT THEN\n" +
			"       	SELECT MIN(LOGID) INTO MIN_ONLINE_LOG_ID FROM FMTP_ONLINE;\n" +
			"			DELETE FROM FMTP_ONLINE WHERE LOGID < (MIN_ONLINE_LOG_ID + CUR_ONLINE_LOG_COUNT - MAX_ONLINE_LOG_COUNT);\n" +
			"		END IF;\n\n" +
			"		SELECT COUNT(*) INTO CUR_STORAGE_LOG_COUNT FROM FMTP_ONLINE;\n" +
			"		IF CUR_STORAGE_LOG_COUNT > MAX_STORAGE_LOG_COUNT THEN\n" +
			"       	SELECT MIN(LOGID) INTO MIN_STORAGE_LOG_ID FROM FMTP_STORAGE;\n" +
			"			DELETE FROM FMTP_STORAGE WHERE LOGID < (MIN_STORAGE_LOG_ID + CUR_STORAGE_LOG_COUNT - MAX_STORAGE_LOG_COUNT);\n" +
			"		END IF;\n" +
			"	END CHECK_LOG_COUNT;\n\n\n" +
			"   --проверка времени хранения логов \n" +
			"   PROCEDURE CHECK_LOG_LIFETIME( OLDEST_STORAGE_LOG_DATE IN FMTP_STORAGE.DATETIME%TYPE)\n" +
			"   AS\n" +
			"   BEGIN\n" +
			"   	DELETE FROM FMTP_STORAGE WHERE DATETIME < OLDEST_STORAGE_LOG_DATE;\n" +
			"   END CHECK_LOG_LIFETIME;\n\n" +
			"END;\n")
}
