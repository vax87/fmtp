package chief_web

import (
	"context"
	"fdps/utils"
	"fmt"
	"log"
	"net/http"
	"time"
)

type httpServer struct {
	http.Server
	shutdownReq chan bool
	done        chan struct{}
	reqCount    uint32
	configPage  *ConfigPage
}

var srv httpServer

//var SettsChan = make(chan settings.Settings, 1)

func Start(done chan struct{}) {
	wsc.load()

	srv = httpServer{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", wsc.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		//shutdownReq: sdc,
		done:       done,
		configPage: new(ConfigPage),
	}
	srv.configPage.initialize("FDPS-FMTP-CHIEF-CONFIG")

	InitEditConfigHandler(utils.ParkingWebConfigPath, "EDIT CONFIG")
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

func (s *httpServer) stop() {
	log.Printf("stoping http server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	err := s.Shutdown(ctx)

	if err != nil {
		log.Printf("shutdown http server error: %v", err)
	}
}

// func SetSetts(setts settings.Settings) {
// 	srv.configPage.Setts = setts
// }
