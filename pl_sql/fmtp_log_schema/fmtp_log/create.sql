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
    procedure check_log_count( max_online_log_count in number,	max_storage_log_count in number	);
    --проверка времени хранения логов
    procedure check_log_lifetime( oldest_storage_log_date in fmtp_storage.datetime%type);
    --обновление состояния канала
    procedure  channel_state_changed( chstate in channel_state );
END;


/
COMMIT;
/

create or replace PACKAGE BODY LOG_PROC_PKG AS
	--проверка кол-ва логов
	procedure check_log_count( max_online_log_count in number,	max_storage_log_count in number	)
	as
		cur_online_log_count   number;
		cur_storage_log_count   number;
       min_online_log_id   fmtp_online.datetime%type;
       min_storage_log_id   fmtp_storage.datetime%type;
	begin
		select count(*) into cur_online_log_count from fmtp_online;
		if cur_online_log_count > max_online_log_count then
       	select min(logid) into min_online_log_id from fmtp_online;
			delete from fmtp_online where logid < (min_online_log_id + cur_online_log_count - max_online_log_count);
		end if;
		select count(*) into cur_storage_log_count from fmtp_storage;
		if cur_storage_log_count > max_storage_log_count then
       	select min(logid) into min_storage_log_id from fmtp_storage;
			delete from fmtp_storage where logid < (min_storage_log_id + cur_storage_log_count - max_storage_log_count);
		end if;
	end check_log_count;
    
    
   --проверка времени хранения логов 
   procedure check_log_lifetime( oldest_storage_log_date in fmtp_storage.datetime%type)
   as
   begin
   	delete from fmtp_storage where datetime < oldest_storage_log_date;
   end check_log_lifetime;
   
   
   
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

