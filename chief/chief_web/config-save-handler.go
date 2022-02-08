package chief_web

import (
	"fdps/utils"
	"net/http"
)

type SaveConfigHandler struct {
	handleURL string
	title     string
}

var SaveConfHandler SaveConfigHandler

func (sch SaveConfigHandler) Path() string {
	return sch.handleURL
}

func (sch SaveConfigHandler) Caption() string {
	return sch.title
}

func (sch SaveConfigHandler) HttpHandler() func(http.ResponseWriter, *http.Request) {
	return saveConfigHandler
}

func InitSaveConfigHandler(handleURL string, title string) {
	SaveConfHandler.handleURL = "/" + handleURL
	SaveConfHandler.title = title
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) {

	srv.configPage.UrlConfig.SettingsURLStr = r.FormValue("SettingsURLStr")
	srv.configPage.UrlConfig.HeartbeatURLStr = r.FormValue("HeartbeatURLStr")

	UrlConfigChan <- srv.configPage.UrlConfig

	http.Redirect(w, r, "/"+utils.FmtpChiefWebLogPath, http.StatusFound)
}
