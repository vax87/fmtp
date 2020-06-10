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

	// dbPort, _ := strconv.Atoi(r.FormValue("DbPort"))

	// srv.configPage.Setts.DbSetts.Hostname = r.FormValue("DbHostname")
	// srv.configPage.Setts.DbSetts.Port = dbPort
	// srv.configPage.Setts.DbSetts.ServiceName = r.FormValue("DbServiceName")
	// srv.configPage.Setts.DbSetts.UserName = r.FormValue("DbUser")
	// srv.configPage.Setts.DbSetts.Password = r.FormValue("DbPassword")

	// wsPort, _ := strconv.Atoi(r.FormValue("WsPort"))
	// srv.configPage.Setts.WsSetts.Port = wsPort

	// srv.configPage.Setts.FirCode = r.FormValue("FirCode")

	// r.ParseForm()
	// if len(r.Form["NeedUpdateANI"]) == 1 {
	// 	srv.configPage.Setts.NeedUpdateANI = r.Form["NeedUpdateANI"][0] == "check"
	// }

	// fmt.Println("NeedUpdateANI", srv.configPage.Setts.NeedUpdateANI)

	// SettsChan <- srv.configPage.Setts

	http.Redirect(w, r, "/"+utils.FmtpChiefWebLogPath, http.StatusFound)
}
