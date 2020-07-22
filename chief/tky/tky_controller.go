package tky

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/fdps"
	"fdps/fmtp/chief_configurator"
	"fdps/utils"
	"fdps/utils/logger"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// TkyController контроллер отвечает за отправкку состояния chief и channel по запросу
type TkyController struct {
	http.Server
	curState StateForTky
}

var tkyCntrl TkyController

// SetAodbProviderStates - задать состояния FDPS провайдеров
func SetAodbProviderStates(aodbStates []fdps.ProviderState) {
	var resStates []ProviderState
	for _, val := range tkyCntrl.curState.ProviderStates {
		if val.ProviderType == fdps.OLDIProvider {
			resStates = append(resStates, val)
		}
	}

	for _, val := range aodbStates {
		resStates = append(resStates,
			ProviderState{
				ProviderID:           val.ProviderID,
				ProviderType:         val.ProviderType,
				ProviderIPs:          val.ProviderIPs,
				ProviderState:        val.ProviderState,
				ProviderStatus:       chief_configurator.ChiefCfg.ProviderStatusById(val.ProviderID),
				ProviderErrorMessage: val.ProviderErrorMessage,
			})
	}

	tkyCntrl.curState.ProviderStates = resStates
}

// SetOldiProviderState - задать состояния FDPS провайдеров
func SetOldiProviderState(oldiStates []fdps.ProviderState) {
	var resStates []ProviderState
	for _, val := range tkyCntrl.curState.ProviderStates {
		if val.ProviderType == fdps.AODBProvider {
			resStates = append(resStates, val)
		}
	}

	for _, val := range oldiStates {
		resStates = append(resStates,
			ProviderState{
				ProviderID:           val.ProviderID,
				ProviderType:         val.ProviderType,
				ProviderIPs:          val.ProviderIPs,
				ProviderState:        val.ProviderState,
				ProviderStatus:       chief_configurator.ChiefCfg.ProviderStatusById(val.ProviderID),
				ProviderErrorMessage: val.ProviderErrorMessage,
			})
	}

	tkyCntrl.curState.ProviderStates = resStates
}

// SetChannelsState - задать состояния FMTP каналов
func SetChannelsState(channelStates []channel_state.ChannelState) {
	tkyCntrl.curState.DaemonStates = tkyCntrl.curState.DaemonStates[:0]

	for _, val := range channelStates {
		tkyCntrl.curState.DaemonStates = append(tkyCntrl.curState.DaemonStates,
			DaemonState{
				DaemonID:    val.ChannelID,
				LocalName:   val.LocalName,
				RemoteName:  val.RemoteName,
				DaemonState: val.DaemonState,
				DaemonType:  chief_configurator.ChiefCfg.ChannelDataTypeById(val.ChannelID),
				FmtpState:   val.FmtpState,
			})
	}
}

// Work -
func Work() {
	wsc.load()

	tkyCntrl.Server = http.Server{
		Addr:         ":" + strconv.Itoa(wsc.Port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.HandleFunc("/"+wsc.Path, tkyStateResponce)
	logger.PrintfInfo("Запускаем HTTP сервер для связи с ТКУ. Порт: %d. Path: %s", wsc.Port, wsc.Path)
	go func() {
		if err := tkyCntrl.ListenAndServe(); err != nil {
			logger.PrintfErr("Ошибка запуска HTTP сервера для связи с ТКУ. Ошибка: %v", err)
		}
	}()
}

func tkyStateResponce(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "POST":
		responceBody := utils.ToJsonByte(tkyCntrl.curState)

		if _, err := w.Write(responceBody); err != nil {
			logger.PrintfErr("Ошибка формирования ответа на запрос ТКУ. Ошибка: %v", err)
		}
	default:
		fmt.Fprintf(w, "Only GET, POST method are supported.")
	}
}
