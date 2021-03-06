package tky

import (
	"fmtp/chief/chief_state"
	"fmtp/configurator"

	"fmt"
	"net/http"
	"strconv"
	"time"

	"lemz.com/fdps/logger"
	"lemz.com/fdps/utils"
)

// TkyController контроллер отвечает за отправкку состояния chief и channel по запросу
type TkyController struct {
	http.Server
}

var tkyCntrl TkyController

func genTkyState() StateForTky {
	var retState StateForTky

	for _, val := range chief_state.CommonChiefState.ChannelStates {
		retState.DaemonStates = append(retState.DaemonStates,
			DaemonState{
				DaemonID:    val.ChannelID,
				LocalName:   val.LocalName,
				RemoteName:  val.RemoteName,
				DaemonState: val.DaemonState,
				DaemonType:  configurator.ChiefCfg.ChannelDataTypeById(val.ChannelID),
				FmtpState:   val.FmtpState,
			})
	}

	for _, val := range chief_state.CommonChiefState.ProviderStates {
		retState.ProviderStates = append(retState.ProviderStates,
			ProviderState{
				ProviderID:           val.ProviderID,
				ProviderType:         val.ProviderType,
				ProviderIPs:          val.ProviderIPs,
				ProviderState:        val.ProviderState,
				ProviderStatus:       configurator.ChiefCfg.ProviderStatusById(val.ProviderID),
				ProviderErrorMessage: val.ProviderErrorMessage,
			})
	}
	return retState
}

func Work() {
	wsc.load()

	tkyCntrl.Server = http.Server{
		Addr:         ":" + strconv.Itoa(wsc.Port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.HandleFunc("/"+wsc.Path, tkyStateResponce)
	logger.PrintfDebug("Запускаем HTTP сервер для связи с ТКУ. Порт: %d. Path: %s", wsc.Port, wsc.Path)
	go func() {
		if err := tkyCntrl.ListenAndServe(); err != nil {
			logger.PrintfErr("Ошибка запуска HTTP сервера для связи с ТКУ. Ошибка: %v", err)
		}
	}()
}

func tkyStateResponce(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "POST":
		responceBody := utils.ToJsonByte(genTkyState())

		if _, err := w.Write(responceBody); err != nil {
			logger.PrintfErr("Ошибка формирования ответа на запрос ТКУ. Ошибка: %v", err)
		}
	default:
		fmt.Fprintf(w, "Only GET, POST method are supported.")
	}
}
