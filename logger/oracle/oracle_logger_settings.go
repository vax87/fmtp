package oracle

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
func (rls *OracleLoggerSettings) equal(otherRls OracleLoggerSettings) (bool, bool) {
	isDbEqual :=
		//rls.NeedWork == otherRls.NeedWork &&
		rls.Hostname == otherRls.Hostname &&
			rls.Port == otherRls.Port &&
			rls.ServiceName == otherRls.ServiceName &&
			rls.UserName == otherRls.UserName &&
			rls.Password == otherRls.Password
	isStorEqual := rls.LogStoreMaxCount == otherRls.LogStoreMaxCount &&
		rls.LogStoreDays == otherRls.LogStoreDays

	return isDbEqual, isStorEqual
}
