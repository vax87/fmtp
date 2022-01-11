DROP USER fmtp CASCADE;
DROP PROFILE fmtp CASCADE;
COMMIT;


create profile fmtp limit
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
create user  fmtp
profile fmtp
identified by fmtppass
account unlock 
default tablespace users 
temporary tablespace temp;  
commit;  
grant resource 			to fmtp;
grant connect 			to fmtp;
grant create session 	to fmtp;
grant alter session 	to fmtp;
grant create any table	to fmtp;
grant create any view	to fmtp;
grant create view		to fmtp;
grant create synonym	to fmtp;

--ORA-01950: нет привилегий на раздел 'USERS'
ALTER USER fmtp quota unlimited on users;
commit;