// logger
package logger_settings

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"fdps/fmtp/logger/file"
	"fdps/fmtp/logger/oracle"
	"fdps/utils/web_sock"
)

// контроллер, отвечающий за считывание настроек из файла и
// передачу настроек соответствующим контроллерам
type SettingsController struct {
	NetSettingsChan          chan web_sock.WebSockServerSettings //
	FileLoggerSettingsChan   chan file.FileLoggerSettings        //
	OracleLoggerSettingsChan chan oracle.OracleLoggerSettings    //
	DoneChan                 chan error                          //
}

// считывание настроек из файла
func (sc *SettingsController) Load() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		sc.DoneChan <- err
		return
	}
	configFileName := dir + filepath.FromSlash("/config/logger_settings.json")
	configFile, err := ioutil.ReadFile(configFileName)

	if err != nil {
		sc.DoneChan <- err
		return
	} else {
		var newFileSettings file.FileLoggerSettings
		if errFS := json.Unmarshal(configFile, &newFileSettings); errFS == nil {
			sc.FileLoggerSettingsChan <- newFileSettings
		} else {
			sc.DoneChan <- errFS
			return
		}
		var newOracleSettings oracle.OracleLoggerSettings
		if errRS := json.Unmarshal(configFile, &newOracleSettings); errRS == nil {
			sc.OracleLoggerSettingsChan <- newOracleSettings
		} else {
			sc.DoneChan <- errRS
			return
		}
		var newNetSettings web_sock.WebSockServerSettings
		if errNS := json.Unmarshal(configFile, &newNetSettings); errNS == nil {
			sc.NetSettingsChan <- newNetSettings
		} else {
			sc.DoneChan <- errNS
			return
		}

		sc.DoneChan <- nil
	}
}
