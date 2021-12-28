package chief_web

import (
	"fdps/fmtp/channel/channel_state"
	"fdps/fmtp/chief/chief_settings"
	"fdps/fmtp/chief/chief_state"
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
	srv.chiefPage.ChannelStates = srv.chiefPage.ChannelStates[:0]

	for _, val := range chief_state.CommonChiefState.ChannelStates {
		channel_st := val

		switch val.DaemonState {
		case channel_state.ChannelStateStopped:
			channel_st.StateColor = StopColor
		case channel_state.ChannelStateError:
			channel_st.StateColor = ErrorColor
		case channel_state.ChannelStateOk:
			channel_st.StateColor = OkColor
		default:
			channel_st.StateColor = DefaultColor
		}
		srv.chiefPage.ChannelStates = append(srv.chiefPage.ChannelStates, channel_st)
	}
	sort.Slice(srv.chiefPage.ChannelStates, func(i, j int) bool {
		return srv.chiefPage.ChannelStates[i].ChannelID < srv.chiefPage.ChannelStates[j].ChannelID
	})

	srv.chiefPage.AodbProviderStates = srv.chiefPage.AodbProviderStates[:0]
	srv.chiefPage.OldiProviderStates = srv.chiefPage.OldiProviderStates[:0]
	for _, val := range chief_state.CommonChiefState.ProviderStates {
		provider_st := val

		switch val.ProviderState {
		case chief_state.StateOk:
			provider_st.StateColor = OkColor
		case chief_state.StateError:
			provider_st.StateColor = ErrorColor
		default:
			provider_st.StateColor = DefaultColor
		}

		if val.ProviderType == chief_settings.OLDIProvider {
			srv.chiefPage.OldiProviderStates = append(srv.chiefPage.OldiProviderStates, provider_st)
		} else {
			srv.chiefPage.AodbProviderStates = append(srv.chiefPage.AodbProviderStates, provider_st)
		}

	}
	sort.Slice(srv.chiefPage.AodbProviderStates, func(i, j int) bool {
		return srv.chiefPage.AodbProviderStates[i].ProviderID < srv.chiefPage.AodbProviderStates[j].ProviderID
	})

	sort.Slice(srv.chiefPage.OldiProviderStates, func(i, j int) bool {
		return srv.chiefPage.OldiProviderStates[i].ProviderID < srv.chiefPage.OldiProviderStates[j].ProviderID
	})

	if err := srv.chiefPage.templ.ExecuteTemplate(w, "ChiefTemplate", srv.chiefPage); err != nil {
		fmt.Println("template ExecuteTemplate TE ERROR", err)
	}
}
