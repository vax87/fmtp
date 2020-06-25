package file

// настройки контроллера записи логов в файл
type FileLoggerSettings struct {
	NeedWork        bool // необходимость работы
	LogFileSizeKb   int  // размер файла в КБ
	LogFolderSizeGb int  // размер папки с логами в ГБ
	LogStoreDays    int  // время хранения логов (дней) в БД
}
