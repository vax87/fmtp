package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"lemz.com/fdps/utils"
)

type Settings struct {
	FmtpCntrlAddress string `json:"FmtpCntrlAddress"`
	FmtpCntrlPort    int    `json:"FmtpCntrlPort"`
	SendIntervalMsec int    `json:"SendIntervalMsec"`
	RecvIntervalMsec int    `json:"RecvIntervalMsec"`

	ImitGenMsgIntervalMsec int `json:"ImitGenMsgIntervalMsec"`
	ImitGenMsgCount        int `json:"ImitGenMsgCount"`

	MetricsGatewayUrl  string `json:"MetricsGatewayUrl"`
	MetricsIntervalSec int    `json:"MetricsIntervalSec"`
	MetricsHostLabel   string `json:"MetricsHostLabel"`
}

var loggerSettingsFile = utils.AppPath() + "/config/imit_settings.json"

// ReadFromFile чтение ранее сохраненных настроек из файла
func (s *Settings) ReadFromFile() error {
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
func (s *Settings) SaveToFile() error {
	if confData, errMrsh := json.Marshal(s); errMrsh != nil {
		return errMrsh
	} else if errWriteSetts := ioutil.WriteFile(loggerSettingsFile, utils.JsonPrettyPrint(confData), os.ModePerm); errWriteSetts != nil {
		return errWriteSetts
	}
	return nil
}

func (s *Settings) setDefault() {
	s.FmtpCntrlAddress = "192.168.1.24"
	s.FmtpCntrlPort = 55566
	s.SendIntervalMsec = 300
	s.RecvIntervalMsec = 300

	s.ImitGenMsgIntervalMsec = 500
	s.ImitGenMsgCount = 10

	s.MetricsGatewayUrl = "http://192.168.1.24:9100"
	s.MetricsIntervalSec = 1
	s.MetricsHostLabel = "127.0.0.1"
}
