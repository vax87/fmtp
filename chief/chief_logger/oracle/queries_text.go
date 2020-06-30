package oracle

import (
	"fmt"

	"fdps/fmtp/chief/chief_logger/common"
)

const (
	onlineLogTableName  = "fmtp_online"
	storageLogTableName = "fmtp_storage"
	onlineLogViewName   = "fmtp_online_vw"
	storageLogViewName  = "fmtp_storage_vw"

	maxTextLen = 2000
)

// создать таблицу с логами.
func oraCreateLogTableQuery(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE %s
		( LogId NUMBER NOT NULL,
		CntrlIP NVARCHAR2(15),
		DaemonID NUMBER,
		LocalName NVARCHAR2(10),
		RemoteName NVARCHAR2(10),
		DataType NVARCHAR2(15),
		Source NVARCHAR2(15),
		Severity NVARCHAR2(15) NOT NULL,
		FmtpType NVARCHAR2(15),
		Direction NVARCHAR2(15),
		DateTime NVARCHAR2(30) NOT NULL,
		Text NVARCHAR2(%d) NOT NULL)`, tableName, maxTextLen)
}

// создать первичный ключ в таблице логов.
func oraCreatePrimaryKeyQuery(tableName string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s_PK PRIMARY KEY(LOGID)", tableName, tableName)
}

// создать последовательность для автоинкремента.
func oraCreateSequenceQuery(tableName string) string {
	return fmt.Sprintf("CREATE SEQUENCE %s_ID_SEQ START WITH 1 NOCACHE ORDER", tableName)
}

// создать триггер для автоинкремента.
func oraCreateTriggerQuery(tableName string) string {
	return fmt.Sprintf(`CREATE OR REPLACE TRIGGER %s_ID_TRG BEFORE 
		INSERT ON %s
		FOR EACH ROW 
		WHEN(NEW.LOGID IS NULL) 
		BEGIN 
		:NEW.LOGID := %s_ID_SEQ.NEXTVAL;
		END;`, tableName, tableName, tableName)
}

// создать представление с логами.
func oraCreateLogViewQuery(viewName string, tableName string) string {
	return fmt.Sprintf(`CREATE OR REPLACE VIEW fmtp_log.%s AS SELECT 
		CntrlIP as cntrlip, 
		DaemonID as daemonid, 
		DataType as datatype, 
		LocalName as localname, 
		RemoteName as remotename, 
		Direction as direction, 
		FmtpType as fmtptype, 
		Severity as severity, 
		Source as source, 
		DateTime as datetime, 
		Text as text 
		from %s`, viewName, tableName)
}

// текст запроса кол-ва строк в таблице tableName.
func oraCheckLogCountQuery(maxOnlineCount int, maxStorageCount int) string {
	return fmt.Sprintf("BEGIN LOG_PROC_PKG.CHECK_LOG_COUNT(%d, %d); COMMIT; END;", maxOnlineCount, maxStorageCount)
}

// текст запроса удаления сообщений старше dateTimeString.
func oraCheckLogLifetimeQuery(dateTimeString string) string {
	return fmt.Sprintf("BEGIN LOG_PROC_PKG.CHECK_LOG_LIFETIME('%s'); COMMIT; END;", dateTimeString)
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

// текст заголовка пакета для удаления старых и лишних логов
func oraPackageQuery() string {
	return `CREATE OR REPLACE PACKAGE LOG_PROC_PKG AS
				--проверка кол-ва логов
			   PROCEDURE CHECK_LOG_COUNT( MAX_ONLINE_LOG_COUNT IN NUMBER,	MAX_STORAGE_LOG_COUNT IN NUMBER	);
				--проверка времени хранения логов
			   PROCEDURE CHECK_LOG_LIFETIME( OLDEST_STORAGE_LOG_DATE IN FMTP_STORAGE.DATETIME%TYPE);
			END;`
}

// текст тела пакета для удаления старых и лишних логов
func oraPackageBodyQuery() string {
	return `CREATE OR REPLACE PACKAGE BODY LOG_PROC_PKG AS
				--проверка кол-ва логов
				PROCEDURE CHECK_LOG_COUNT( MAX_ONLINE_LOG_COUNT IN NUMBER,	MAX_STORAGE_LOG_COUNT IN NUMBER	)
				AS
					CUR_ONLINE_LOG_COUNT   NUMBER;
					CUR_STORAGE_LOG_COUNT   NUMBER;
			       MIN_ONLINE_LOG_ID   FMTP_ONLINE.DATETIME%TYPE;
			       MIN_STORAGE_LOG_ID   FMTP_STORAGE.DATETIME%TYPE;
				BEGIN
					SELECT COUNT(*) INTO CUR_ONLINE_LOG_COUNT FROM FMTP_ONLINE;
					IF CUR_ONLINE_LOG_COUNT > MAX_ONLINE_LOG_COUNT THEN
			       	SELECT MIN(LOGID) INTO MIN_ONLINE_LOG_ID FROM FMTP_ONLINE;
						DELETE FROM FMTP_ONLINE WHERE LOGID < (MIN_ONLINE_LOG_ID + CUR_ONLINE_LOG_COUNT - MAX_ONLINE_LOG_COUNT);
					END IF;
					SELECT COUNT(*) INTO CUR_STORAGE_LOG_COUNT FROM FMTP_ONLINE;
					IF CUR_STORAGE_LOG_COUNT > MAX_STORAGE_LOG_COUNT THEN
			       	SELECT MIN(LOGID) INTO MIN_STORAGE_LOG_ID FROM FMTP_STORAGE;
						DELETE FROM FMTP_STORAGE WHERE LOGID < (MIN_STORAGE_LOG_ID + CUR_STORAGE_LOG_COUNT - MAX_STORAGE_LOG_COUNT);
					END IF;
				END CHECK_LOG_COUNT;
			   --проверка времени хранения логов 
			   PROCEDURE CHECK_LOG_LIFETIME( OLDEST_STORAGE_LOG_DATE IN FMTP_STORAGE.DATETIME%TYPE)
			   AS
			   BEGIN
			   	DELETE FROM FMTP_STORAGE WHERE DATETIME < OLDEST_STORAGE_LOG_DATE;
			   END CHECK_LOG_LIFETIME;
			END;`
}
