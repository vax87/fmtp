package redis_cntrl

// RedisCntrlSettings - настройки контроллера отправки логов в БД Redis
type RedisCntrlSettings struct {
	Hostname string `json:"DbHostname"` // адрес/название хоста
	Port     int    `json:"DbPort"`     // порт подключения к БД
	DbId     int    `json:"DbId"`       // идентификатор БД
	UserName string `json:"DbUser"`     // пользователь БД
	Password string `json:"DbPassword"` // пароль для подключения к БД

	MaxReadCount int `json:"MaxReadCount"` // максимальное кол-во логов, считываемых за один раз
}

// сравнение настроек в части настроек БД
func (s *RedisCntrlSettings) equalDb(otherRls RedisCntrlSettings) bool {
	return s.Hostname == otherRls.Hostname &&
		s.Port == otherRls.Port &&
		s.UserName == otherRls.UserName &&
		s.Password == otherRls.Password
}
