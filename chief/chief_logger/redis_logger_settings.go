package chief_logger

// RedisLoggerSettings - настройки контроллера отправки логов в БД Redis
type RedisLoggerSettings struct {
	Hostname string `json:"DbHostname"` // адрес/название хоста
	Port     int    `json:"DbPort"`     // порт подключения к БД
	DbId     int    `json:"DbId"`       // идентификатор БД
	UserName string `json:"DbUser"`     // пользователь БД
	Password string `json:"DbPassword"` // пароль для подключения к БД

	StreamMaxCount int64 `json:"StreamMaxCount"` // максимальное число логов в потоке
	MaxSendCount   int   `json:"MaxSendCount"`   // максимальное кол-во логов, отправляемых за один раз
}

// сравнение настроек в части настроек БД
func (s *RedisLoggerSettings) equalDb(otherRls RedisLoggerSettings) bool {
	return s.Hostname == otherRls.Hostname &&
		s.Port == otherRls.Port &&
		s.UserName == otherRls.UserName &&
		s.Password == otherRls.Password
}
