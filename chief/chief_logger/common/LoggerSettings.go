package common

import (
	"encoding/json"
	"fdps/utils"
	"fmt"
	"io/ioutil"
	"os"
)

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

var loggerSettingsFile = utils.AppPath() + "/config/logger_settings.json"

// ReadFromFile чтение ранее сохраненных настроек из файла
func (s *LoggerSettings) ReadFromFile() error {
	if data, err := ioutil.ReadFile(loggerSettingsFile); err != nil {
		return err
	} else if err := json.Unmarshal(data, &s); err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// SaveToFile сохранение настроек в файл
func (s *LoggerSettings) SaveToFile() error {
	if confData, err := json.Marshal(s); err != nil {
		return err
	} else if err := ioutil.WriteFile(loggerSettingsFile, utils.JsonPrettyPrint(confData), os.ModePerm); err != nil {
		return err
	}
	return nil
}

// DefaultInit настройки по умолчанию
func (s *LoggerSettings) DefaultInit() {
	s.FileSizeKB = 500
	s.FolderSizeGB = 10
	s.LoggerPort = 54354
	s.DbServiceName = "xe"
	s.DbHostname = "192.168.1.24"
	s.DbMaxLogStoreCount = 100000
	s.DbPassword = "log"
	s.DbPort = 1521
	s.DbStoreDays = 30
	s.DbUser = "fmtp_log"
}
