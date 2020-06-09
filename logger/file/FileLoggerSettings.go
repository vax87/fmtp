package file

// настройки контроллера записи логов в файл
type FileLoggerSettings struct {
	NeedWork        bool `json:"LoggerNeedWork"`    // необходимость работы
	LogFileSizeKb   int  `json:"FileSizeKB"`   // размер файла в КБ
	LogFolderSizeGb int  `json:"FolderSizeGB"` // размер папки с логами в ГБ
	LogStoreDays    int  `json:"DbStoreDays"`       // время хранения логов (дней) в БД
}
