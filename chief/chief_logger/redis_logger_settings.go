package chief_logger

// RedisLoggerSettings - настройки контроллера отправки логов в БД Redis
type RedisLoggerSettings struct {
	Hostname string `json:"RedisHostname"` // адрес/название хоста
	Port     int    `json:"RediPort"`      // порт подключения к БД
	DbId     int    `json:"RedisDbId"`     // идентификатор БД
	UserName string `json:"RedisUser"`     // пользователь БД
	Password string `json:"RedisPassword"` // пароль для подключения к БД

	StreamMaxCount int64 `json:"RedisStreamMaxCount"` // максимальное число логов в потоке
	MaxSendCount   int   `json:"RedisMaxSendCount"`   // максимальное кол-во логов, отправляемых за один раз
}

// сравнение настроек в части настроек БД
func (s *RedisLoggerSettings) equalDb(otherRls RedisLoggerSettings) bool {
	return s.Hostname == otherRls.Hostname &&
		s.Port == otherRls.Port &&
		s.UserName == otherRls.UserName &&
		s.Password == otherRls.Password
}
