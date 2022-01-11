--run as fdps
--
alter table oldichannel add fmtpstate nvarchar2(15) NULL;

grant select, update, insert    on oldichannel  to fmtp_log;

commit;