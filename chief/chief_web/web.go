package chief_web

import (
	"fdps/fmtp/chief_configurator/configurator_urls"
	"fdps/utils"
	"fmt"
	"log"
	"net/http"
	"time"
)

type httpServer struct {
	http.Server

	done       chan struct{}
	configPage *ConfigPage
	chiefPage  *ChiefPage
}

var srv httpServer

var UrlConfigChan = make(chan configurator_urls.ConfiguratorUrls, 1)

func Start(done chan struct{}) {
	wsc.load()

	srv = httpServer{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", wsc.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		done:       done,
		configPage: new(ConfigPage),
		chiefPage:  new(ChiefPage),
	}
	srv.configPage.initialize("FDPS-FMTP-CHIEF-CONFIG")
	srv.chiefPage.initialize("FDPS-FMTP-CHIEF")
	InitChiefChannelsHandler(utils.FmtpChiefWebPath, "CHIEF")
	utils.AppendHandler(ChiefHdl)

	InitEditConfigHandler(utils.FmtpChiefWebConfigPath, "EDIT CONFIG")
	utils.AppendHandler(EditConfHandler)
	InitSaveConfigHandler("saveConfig", "SAVE PARKING")
	utils.AppendHandler(SaveConfHandler)

	for _, h := range utils.HandlerList {
		http.HandleFunc(h.Path(), h.HttpHandler())
	}

	log.Printf("listening on %d port started", wsc.Port)

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Printf("listen and serve error : %v", err)
		}
	}()
}

func SetUrlConfig(urlConfig configurator_urls.ConfiguratorUrls) {
	srv.configPage.UrlConfig = urlConfig
}
