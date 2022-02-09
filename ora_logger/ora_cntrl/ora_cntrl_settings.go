package ora_cntrl

import "strconv"

// OraCntrlSettings - настройки контроллера записи логов в БД
type OraCntrlSettings struct {
	Hostname    string `json:"DbHostname"`    // адрес/название хоста
	Port        int    `json:"DbPort"`        // порт подключения к БД
	ServiceName string `json:"DbServiceName"` // название сервиса БД
	UserName    string `json:"DbUser"`        // пользователь БД
	Password    string `json:"DbPassword"`    // пароль для подключения к БД

	LogStoreMaxCount int `json:"DbMaxLogStoreCount"` // максимальное число хранимых логов (шт)
	LogStoreDays     int `json:"DbStoreDays"`        // время хранения логов (дней)
}

// сравнение настроек в части настроек БД и настроек хранения логов
func (s *OraCntrlSettings) equal(other OraCntrlSettings) (bool, bool) {
	isDbEqual := s.Hostname == other.Hostname &&
		s.Port == other.Port &&
		s.ServiceName == other.ServiceName &&
		s.UserName == other.UserName &&
		s.Password == other.Password
	isStorEqual := s.LogStoreMaxCount == other.LogStoreMaxCount &&
		s.LogStoreDays == other.LogStoreDays

	return isDbEqual, isStorEqual
}

// ConnString - строка подключения к БД в формате
// user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))
func (s *OraCntrlSettings) ConnString() string {
	return s.UserName + "/" +
		s.Password + "@" +
		"(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=" + s.Hostname +
		")(PORT=" + strconv.Itoa(s.Port) +
		")))(CONNECT_DATA=(SERVICE_NAME=" + s.ServiceName + ")))"
}
