package chief_web

import (
	"fdps/utils"
	"fdps/utils/logger"
)

type webServerConf struct {
	Port uint16 `json:"port"`
}

var webConfigName = utils.AppPath() + "/config/fdps-fmtp-chief-web-config.json"

var wsc webServerConf

func (c *webServerConf) load() {
	if errRead := utils.ReadFromFile(webConfigName, c); errRead != nil {
		logger.PrintfErr("Ошибка чтения конфигурации Web страницы из файла. Файл: %s. Ошибка: %s", webConfigName, errRead)
		c.setDefault()
	} else {
		if c.Port <= 1024 {
			logger.PrintfErr("Невалидное значение сетевого порта для Web страницы. Порт: %d", c.Port)
			c.setDefault()
		}
	}
}

func (c *webServerConf) save() {
	if errWrite := utils.SaveToFile(webConfigName, c); errWrite != nil {
		logger.PrintfErr("Ошибка записи конфигурации Web страницы в файл. Файл: %s. Ошибка: %s", webConfigName, errWrite)
	}
}

func (c *webServerConf) setDefault() {
	c.Port = utils.FmtpChiefWebLogPort
	c.save()
}
