package chief_settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"fdps/fmtp/channel/channel_settings"
	"fdps/fmtp/chief/chief_logger/common"
	"fdps/fmtp/chief/fdps"
	"fdps/utils"
)

// Settings настройки контроллера(chief)
type ChiefSettings struct {
	IPAddr               string `json:"ControllerIP"`         // IP адрес контроллера
	CntrlID              int    `json:"ControllerID"`         // идентификатор контроллера
	Timestamp            string `json:"ConfigTimestamp"`      // метка времени
	ChannelsPort         int    `json:"DaemonsPort"`          // TCP порт для связи с демонами.
	OldiProviderPort     int    `json:"OldiProviderPort"`     // TCP порт для связи с плановым сервисом (OLDI).
	OldiProviderEncoding string `json:"OldiProviderEncoding"` // кодировка сообщений при общении с провайдером OLDI ("Windows-1251" | "UTF-8")
	AodbProviderPort     int    `json:"AodbProviderPort"`     // TCP порт для связи с плановым сервисом (AODB).
	DockerRegistry       string `json:"DockerRegistry"`       // репозиторий с docker образами каналовы

	LoggerSetts    common.LoggerSettings              `json:"LoggerSettings"`
	ChannelSetts   []channel_settings.ChannelSettings `json:"FmtpDaemons"`
	ProvidersSetts []fdps.ProviderSettings            `json:"Providers"`
	IsInitialised  bool                               `json:"-"` // признак инициализации настроек (либо получены от конфигуратора, либо считаны из файла)
}

var chiefSettingsFile = utils.AppPath() + "/config/fmtp_settings.json"

// ReadFromFile чтение ранее сохраненных настроек из файла
func (s *ChiefSettings) ReadFromFile() error {
	data, err := ioutil.ReadFile(chiefSettingsFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

// SaveToFile сохранение настроек в файл
func (s *ChiefSettings) SaveToFile() error {
	if confData, errMrsh := json.Marshal(s); errMrsh != nil {
		return errMrsh
	} else if errWriteSetts := ioutil.WriteFile(chiefSettingsFile, utils.JsonPrettyPrint(confData), os.ModePerm); errWriteSetts != nil {
		return errWriteSetts
	}
	return nil
}

// ProviderStatusById - статус (master|slave) провайдера по идентификатору
func (s *ChiefSettings) ProviderStatusById(idProv int) string {
	for _, val := range s.ProvidersSetts {
		if val.ID == idProv {
			return val.Status
		}
	}
	return "primary"
}

// ChannelDataTypeById - тип данных (OLDI|AODB) канала по идентификатору
func (s *ChiefSettings) ChannelDataTypeById(idChan int) string {
	for _, val := range s.ChannelSetts {
		if val.Id == idChan {
			return val.DataType
		}
	}
	return "OLDI"
}
