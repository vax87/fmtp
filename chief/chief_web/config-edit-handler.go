package chief_web

import (
	"fmt"
	"net/http"
)

type EditConfigHandler struct {
	handleURL string
	title     string
}

var EditConfHandler EditConfigHandler

func (cw EditConfigHandler) Path() string {
	return cw.handleURL
}

func (cw EditConfigHandler) Caption() string {
	return cw.title
}

func (ch EditConfigHandler) HttpHandler() func(http.ResponseWriter, *http.Request) {
	return editConfigHandler
}

func InitEditConfigHandler(handleURL string, title string) {
	EditConfHandler.handleURL = "/" + handleURL
	EditConfHandler.title = title
}

func editConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := srv.configPage.editTempl.ExecuteTemplate(w, "Config", srv.configPage); err != nil {
		fmt.Println("template ExecuteTemplate TE ERROR", err)
	}
}
