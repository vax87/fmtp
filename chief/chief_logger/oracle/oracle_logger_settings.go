package oracle

import "strconv"

//"encoding/json"
//"io/ioutil"
//"os"
//"path/filepath"

// настройки контроллера записи логов в БД
type OracleLoggerSettings struct {
	//NeedWork			bool	`json:"LoggerNeedWork"`			// необходимость работы
	Hostname    string `json:"DbHostname"`    // адрес/название хоста
	Port        int    `json:"DbPort"`        // порт подключения к БД
	ServiceName string `json:"DbServiceName"` // название сервиса БД
	UserName    string `json:"DbUser"`        // пользователь БД
	Password    string `json:"DbPassword"`    // пароль для подключения к БД

	LogStoreMaxCount int `json:"DbMaxLogStoreCount"` // максимальное число хранимых логов (шт)
	LogStoreDays     int `json:"DbStoreDays"`        // время хранения логов (дней)
}

// сравнение настроек в части настроек БД и настроек хранения логов
func (s *OracleLoggerSettings) equal(otherRls OracleLoggerSettings) (bool, bool) {
	isDbEqual :=
		//rls.NeedWork == otherRls.NeedWork &&
		s.Hostname == otherRls.Hostname &&
			s.Port == otherRls.Port &&
			s.ServiceName == otherRls.ServiceName &&
			s.UserName == otherRls.UserName &&
			s.Password == otherRls.Password
	isStorEqual := s.LogStoreMaxCount == otherRls.LogStoreMaxCount &&
		s.LogStoreDays == otherRls.LogStoreDays

	return isDbEqual, isStorEqual
}

// ConnString - строка подключения к БД в формате
// user/pass@(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=hostname)(PORT=port)))(CONNECT_DATA=(SERVICE_NAME=sn)))
func (s *OracleLoggerSettings) ConnString() string {
	return s.UserName + "/" +
		s.Password + "@" +
		"(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=tcp)(HOST=" + s.Hostname +
		")(PORT=" + strconv.Itoa(s.Port) +
		")))(CONNECT_DATA=(SERVICE_NAME=" + s.ServiceName + ")))"

}
