package channel_state

import "time"

const (
	ChannelStateOk      = "ok"
	ChannelStateStopped = "stopped"
	ChannelStateError   = "error"

	StateSendInterval = 1 * time.Second
)

type ChannelState struct {
	ChannelID   int    `json:"DaemonID"`    // идентификатор канала *Не переменовывать в ChannelId
	LocalName   string `json:"LocalName"`   // локальный ATC
	RemoteName  string `json:"RemoteName"`  // удаленный ATC
	DaemonState string `json:"DaemonState"` // состояние канала *Не переменовывать в ChannelState
	FmtpState   string `json:"FmtpState"`   // FMTP состояние канала
	ChannelURL  string `json:"ChannelURL"`  // URL web странички канала
	StateColor  string `json:"-"`
}
