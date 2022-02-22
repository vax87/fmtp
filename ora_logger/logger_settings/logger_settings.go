package logger_settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"lemz.com/fdps/utils"
)

type LoggerSettings struct {
	IsInitialised bool
	IPAddr        string `json:"LoggerIP"`
	LoggerID      string `json:"LoggerID"`
	Timestamp     string `json:"LoggerConfigTimestamp"`

	OraHostname    string `json:"OraHostname"`
	OraPort        int    `json:"OraPort"`
	OraServiceName string `json:"OraServiceName"`
	OraUser        string `json:"OraUser"`
	OraPassword    string `json:"OraPassword"`

	OraMaxLogStoreCount int `json:"OraMaxLogStoreCount"`
	OraStoreDays        int `json:"OraStoreDays"`

	RedisHostname string `json:"RedisHostname"`
	RedisPort     int    `json:"RedisPort"`
	RedisDbId     int    `json:"RedisId"`
	RedisUserName string `json:"RedisUser"`
	RedisPassword string `json:"RedisPassword"`

	RedisMaxReadCount int `json:"RedisMaxReadCount"`

	MetricsIntervalSec int    `json:"MetricsIntervalSec"`
	MetricsGatewayUrl  string `json:"MetricsGatewayUrl"`
}

var loggerSettingsFile = utils.AppPath() + "/config/logger_settings.json"

// ReadFromFile чтение ранее сохраненных настроек из файла
func (s *LoggerSettings) ReadFromFile() error {
	data, err := ioutil.ReadFile(loggerSettingsFile)
	if err != nil {
		s.setDefault()
		s.SaveToFile()
		return err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// SaveToFile сохранение настроек в файл
func (s *LoggerSettings) SaveToFile() error {
	if confData, errMrsh := json.Marshal(s); errMrsh != nil {
		return errMrsh
	} else if errWriteSetts := ioutil.WriteFile(loggerSettingsFile, utils.JsonPrettyPrint(confData), os.ModePerm); errWriteSetts != nil {
		return errWriteSetts
	}
	return nil
}

func (s *LoggerSettings) setDefault() {
	s.IsInitialised = false
	s.IPAddr = "127.0.0.1"
	s.LoggerID = "-1"
	s.Timestamp = ""

	s.OraHostname = "192.168.1.30"
	s.OraPort = 1521
	s.OraServiceName = "metplan"
	s.OraUser = "fmtp_log"
	s.OraPassword = "log"

	s.OraMaxLogStoreCount = 1000000
	s.OraStoreDays = 30

	s.RedisHostname = "192.168.1.24"
	s.RedisPort = 6389
	s.RedisDbId = 0
	s.RedisUserName = ""
	s.RedisPassword = ""

	s.RedisMaxReadCount = 100

	s.MetricsIntervalSec = 1
	s.MetricsGatewayUrl = "http://192.168.1.24:9100"
}
