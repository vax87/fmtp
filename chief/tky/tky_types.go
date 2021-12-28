package tky

type StateForTky struct {
	DaemonStates   []DaemonState
	ProviderStates []ProviderState
}

type DaemonState struct {
	DaemonID    int
	LocalName   string
	RemoteName  string
	DaemonState string
	DaemonType  string
	FmtpState   string
}

type ProviderState struct {
	ProviderID           int
	ProviderType         string
	ProviderIPs          []string
	ProviderState        string
	ProviderStatus       string
	ProviderErrorMessage string
}
