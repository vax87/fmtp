package configurator_urls

import (
	"net/url"

	//"fdps/go_utils/logger"
	"fdps/utils"

	"lemz.com/fdps/logger"
)

// ConfiguratorUrls URL для взаимодействия конфигуратора.
type ConfiguratorUrls struct {
	HeartbeatURLStr string   `json:"HeartbeatUrl"` // url, на который отправляется сообщение о состоянии
	HeartbeatURL    *url.URL `json:"-"`
	SettingsURLStr  string   `json:"SettingsUrl"` // url, на который отправляется запрос настроек
	SettingsURL     *url.URL `json:"-"`
}

var confFilePath = utils.AppPath() + "/config/config_urls.json"

func (cu *ConfiguratorUrls) ReadFromFile() {
	if errRead := utils.ReadFromFile(confFilePath, cu); errRead != nil {
		logger.PrintfErr("Ошибка чтения файла конфигураций URL. Ошибка: %v.", errRead)
	}

	var urlErr error
	if cu.HeartbeatURL, urlErr = url.ParseRequestURI(cu.HeartbeatURLStr); urlErr != nil {
		logger.PrintfErr("Неверное значение URL отправки состояния конфигуратору. URL: %s. Ошибка: %v", cu.SettingsURLStr, urlErr)
	}
	if cu.SettingsURL, urlErr = url.ParseRequestURI(cu.SettingsURLStr); urlErr != nil {
		logger.PrintfErr("Неверное значение URL запроса настроек у конфигуратора. URL: %s. Ошибка: %v", cu.SettingsURLStr, urlErr)
	}
}

func (cu *ConfiguratorUrls) SaveToFile() {
	if errWrite := utils.SaveToFile(confFilePath, cu); errWrite != nil {
		logger.PrintfErr("Ошибка сохранения конфигурации URL в файл. Ошибка: %v", errWrite)
	}
}
