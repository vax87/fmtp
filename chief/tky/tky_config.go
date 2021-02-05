package tky

import (
	"fdps/utils"
	"fdps/utils/logger"
)

type webServerConf struct {
	Port int    `json:"port"`
	Path string `json:"path"`
}

var webConfigName = utils.AppPath() + "/config/tky-web-config.json"

var wsc webServerConf

func (c *webServerConf) load() {
	if errRead := utils.ReadFromFile(webConfigName, c); errRead != nil {
		logger.PrintfErr("Ошибка чтения конфигурации Web страницы из файла. Файл: %s. Ошибка: %s", webConfigName, errRead)
		c.setDefault()
	}
}

func (c *webServerConf) save() {
	if errWrite := utils.SaveToFile(webConfigName, c); errWrite != nil {
		logger.PrintfErr("Ошибка записи конфигурации Web страницы в файл. Файл: %s. Ошибка: %s", webConfigName, errWrite)
	}
}

func (c *webServerConf) setDefault() {
	c.Port = utils.FmtpTkyWebPort
	c.Path = utils.FmtpTkyWebPath
	c.save()
}
