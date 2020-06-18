package chief_web

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/fdps"
	"fmt"
	"net/http"
	"sort"
)

const (
	OkColor      = "#DFF7DE"
	ErrorColor   = "#F2C4CA"
	StopColor    = "#F4EDBA"
	DefaultColor = "#EAECEE"
)

type ChiefHandler struct {
	handleURL string
	title     string
}

var ChiefHdl ChiefHandler

func (cw ChiefHandler) Path() string {
	return cw.handleURL
}

func (cw ChiefHandler) Caption() string {
	return cw.title
}

func (cw ChiefHandler) HttpHandler() func(http.ResponseWriter, *http.Request) {
	return cw.handlerMain
}

func InitChiefChannelsHandler(handleURL string, title string) {
	ChiefHdl.handleURL = "/" + handleURL
	ChiefHdl.title = title
}

func (cw *ChiefHandler) handlerMain(w http.ResponseWriter, r *http.Request) {
	for idx, nn := range srv.chiefPage.ChannelStates {
		switch nn.DaemonState {
		case channel_state.ChannelStateStopped:
			srv.chiefPage.ChannelStates[idx].StateColor = StopColor
		case channel_state.ChannelStateError:
			srv.chiefPage.ChannelStates[idx].StateColor = ErrorColor
		case channel_state.ChannelStateOk:
			srv.chiefPage.ChannelStates[idx].StateColor = OkColor
		default:
			srv.chiefPage.ChannelStates[idx].StateColor = DefaultColor
		}
	}
	sort.Slice(srv.chiefPage.ChannelStates, func(i, j int) bool {
		return srv.chiefPage.ChannelStates[i].ChannelID < srv.chiefPage.ChannelStates[j].ChannelID
	})

	for idx, nn := range srv.chiefPage.AodbProviderStates {
		switch nn.ProviderState {
		case fdps.ProviderStateOk:
			srv.chiefPage.AodbProviderStates[idx].StateColor = OkColor
		case fdps.ProviderStateError:
			srv.chiefPage.AodbProviderStates[idx].StateColor = ErrorColor
		default:
			srv.chiefPage.AodbProviderStates[idx].StateColor = DefaultColor
		}
	}
	sort.Slice(srv.chiefPage.AodbProviderStates, func(i, j int) bool {
		return srv.chiefPage.AodbProviderStates[i].ProviderID < srv.chiefPage.AodbProviderStates[j].ProviderID
	})

	for idx, nn := range srv.chiefPage.OldiProviderStates {
		switch nn.ProviderState {
		case fdps.ProviderStateOk:
			srv.chiefPage.OldiProviderStates[idx].StateColor = OkColor
		case fdps.ProviderStateError:
			srv.chiefPage.OldiProviderStates[idx].StateColor = ErrorColor
		default:
			srv.chiefPage.OldiProviderStates[idx].StateColor = DefaultColor
		}
	}
	sort.Slice(srv.chiefPage.OldiProviderStates, func(i, j int) bool {
		return srv.chiefPage.OldiProviderStates[i].ProviderID < srv.chiefPage.OldiProviderStates[j].ProviderID
	})

	if err := srv.chiefPage.templ.ExecuteTemplate(w, "ChiefTemplate", srv.chiefPage); err != nil {
		fmt.Println("template ExecuteTemplate TE ERROR", err)
	}
}
