DROP TABLE fmtp_online;
DROP SEQUENCE fmtp_online_ID_SEQ;
/
COMMIT;
/


CREATE TABLE fmtp_online
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
		Text NVARCHAR2(2000) NOT NULL,
		CONSTRAINT fmtp_online_PK PRIMARY KEY(LogId));

/

CREATE SEQUENCE fmtp_online_ID_SEQ START WITH 1 NOCACHE ORDER;

/

CREATE OR REPLACE TRIGGER fmtp_online_ID_TRG BEFORE 
		INSERT ON fmtp_online
		FOR EACH ROW 
		WHEN(NEW.LOGID IS NULL) 
		BEGIN 
		:NEW.LOGID := fmtp_online_ID_SEQ.NEXTVAL;
		END;
        
/
COMMIT;
/


DROP TABLE fmtp_storage;
DROP SEQUENCE fmtp_storage_ID_SEQ;
/
COMMIT;
/


CREATE TABLE fmtp_storage
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
		Text NVARCHAR2(2000) NOT NULL,
		CONSTRAINT fmtp_storage_PK PRIMARY KEY(LogId));

/

CREATE SEQUENCE fmtp_storage_ID_SEQ START WITH 1 NOCACHE ORDER;

/

CREATE OR REPLACE TRIGGER fmtp_storage_ID_TRG BEFORE 
		INSERT ON fmtp_storage
		FOR EACH ROW 
		WHEN(NEW.LOGID IS NULL) 
		BEGIN 
		:NEW.LOGID := fmtp_storage_ID_SEQ.NEXTVAL;
		END;
        
/
COMMIT;
/


CREATE OR REPLACE VIEW fmtp_online_vw AS SELECT 
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
		from fmtp_online;

/
COMMIT;
/

CREATE OR REPLACE VIEW fmtp_storage_vw AS SELECT 
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
		from fmtp_storage;

/
COMMIT;
/

create or replace PACKAGE LOG_PROC_PKG AS
    
    -- состояние канала
    type channel_state is record
    (
		m_localname     nvarchar2(10)   := '',
		m_remotename    nvarchar2(10)   := '',
		m_workingstate  nvarchar2(10)   := '',
        m_fmtpstate     nvarchar2(10)   := ''
    );
    
    --проверка кол-ва логов
    PROCEDURE CHECK_LOG_COUNT( MAX_ONLINE_LOG_COUNT IN NUMBER,	MAX_STORAGE_LOG_COUNT IN NUMBER	);
    --проверка времени хранения логов
    PROCEDURE CHECK_LOG_LIFETIME( OLDEST_STORAGE_LOG_DATE IN FMTP_STORAGE.DATETIME%TYPE);
    --обновление состояния канала
    procedure  channel_state_changed( chstate in channel_state );
END;


/
COMMIT;
/

create or replace PACKAGE BODY LOG_PROC_PKG AS
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
   
   
   
    --обновление состояния канала
    procedure  channel_state_changed( chstate in channel_state )
    is        
    begin            
    
        update fdps.oldichannel 
            set fmtpstate = chstate.m_fmtpstate
        where 
            cid = chstate.m_remotename or
            remoteid = chstate.m_remotename;
            
   end channel_state_changed;
END;



/
COMMIT;
/

