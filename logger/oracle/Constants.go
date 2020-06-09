package oracle


const (
	onlineLogTableName  = "fmtp_online"
	storageLogTableName = "fmtp_storage"
	onlineLogViewName   = "fmtp_online_vw"
	storageLogViewName  = "fmtp_storage_vw"
)

//const avgLogMessageSize = 700                     // средний размер одного сообщения логов (байт).
const logContainerSize = 200 * 1024  * 1024 / 700	// размер контейнера с сообщениями (200 МБ / avgLogMessageSize_).
const roachMaxInsertCount = 100						// максимальное кол-во вставляемых элементов в одном INSERT (cockroach)
const oraMaxInsertCount = 100					    // максимальное кол-во вставляемых элементов в одном INSERT (oracle)
const insertCountCheck = 1000						// кол-во запросов INSERT в БД, после чего следует проверить кол-во хранимых сообщений
const onlineLogMaxCount = 1000						// кол-во хранимых логов в таблице онлайн сообщений
