package chief_configurator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

// ConfiguratorUrls URL для взаимодействия конфигуратора.
type ConfiguratorUrls struct {
	HeartbeatURLStr string   `json:"HeartbeatUrl"` // url, на который отправляется сообщение о состоянии
	HeartbeatURL    *url.URL `json:"",omitempty`
	SettingsURLStr  string   `json:"SettingsUrl"` // url, на который отправляется запрос настроек
	SettingsURL     *url.URL `json:"",omitempty`
}

// чтение настроек URL из файла
func (cu *ConfiguratorUrls) Read() error {
	var confFilePath string // путь к файлу с настройками URL
	if ex, err := os.Executable(); err == nil {
		confFilePath = filepath.Dir(ex) + "/config/config_urls.json"
	} else {
		return fmt.Errorf("Ошибка чтения файла конфигураций URL. Ошибка: %s.", err.Error())
	}

	if configFile, err := ioutil.ReadFile(confFilePath); err == nil {
		if err := json.Unmarshal(configFile, cu); err != nil {
			return fmt.Errorf("Ошибка разбора файла конфигураций URL. Ошибка: %s.", err.Error())
		}
	} else {
		return fmt.Errorf("Ошибка чтения файла конфигураций URL. Ошибка: %s.", err.Error())
	}

	var urlErr error
	if cu.HeartbeatURL, urlErr = url.ParseRequestURI(cu.HeartbeatURLStr); urlErr != nil {
		return fmt.Errorf("Не верное значение URL отправки состояния конфигуратору.Значение URL: %s.", cu.SettingsURLStr)
	}
	if cu.SettingsURL, urlErr = url.ParseRequestURI(cu.SettingsURLStr); urlErr != nil {
		return fmt.Errorf("Не верное значение URL запроса настроек у конфигуратора. Значение URL: %s.", cu.SettingsURLStr)
	}
	return nil
}
