package common

// LoggerSettings настройки сервиса логирования
type LoggerSettings struct {
	FileSizeKB         int    `json:"FileSizeKB"`
	FolderSizeGB       int    `json:"FolderSizeGB"`
	LoggerPort         int    `json:"LoggerPort"`
	DbServiceName      string `json:"DbServiceName"`
	DbHostname         string `json:"DbHostname"`
	DbMaxLogStoreCount int    `json:"DbMaxLogStoreCount`
	DbPassword         string `json:"DbPassword"`
	DbPort             int    `json:"DbPort"`
	DbStoreDays        int    `json:"DbStoreDays"`
	DbUser             string `json:"DbUser"`
}
