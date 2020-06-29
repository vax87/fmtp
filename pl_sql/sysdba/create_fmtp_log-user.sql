DROP USER fmtp_log CASCADE;
DROP PROFILE fmtp_log CASCADE;
COMMIT;

--error: ORA-65096: invalid common user or role name in oracle
alter session set "_ORACLE_SCRIPT"=true;  

create profile fmtp_log limit
cpu_per_session default
cpu_per_call default
connect_time default
idle_time default
sessions_per_user default
logical_reads_per_session default
logical_reads_per_call default
private_sga default
composite_limit default
password_life_time unlimited
password_grace_time default
password_reuse_max default
password_reuse_time default
password_verify_function default
failed_login_attempts unlimited
password_lock_time default;
commit;

ALTER PROFILE DEFAULT LIMIT PASSWORD_LIFE_TIME UNLIMITED;
commit;
create user  fmtp_log
profile fmtp_log
identified by log
account unlock 
default tablespace users 
temporary tablespace temp;  
commit;  
grant resource 			to fmtp_log;
grant connect 			to fmtp_log;
grant create session 	to fmtp_log;
grant alter session 	to fmtp_log;
grant create any table	to fmtp_log;
grant create any view	to fmtp_log;
grant create view		to fmtp_log;
grant create synonym	to fmtp_log;
commit;

--ORA-01950: нет привилегий на раздел 'USERS'
ALTER USER fmtp_log quota unlimited on users;
