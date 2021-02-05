package tky

// StateForTky -
type StateForTky struct {
	//vendor, product string
	DaemonStates   []DaemonState
	ProviderStates []ProviderState
}

// DaemonState -
type DaemonState struct {
	DaemonID    int
	LocalName   string
	RemoteName  string
	DaemonState string
	DaemonType  string
	FmtpState   string
}

// ProviderState -
type ProviderState struct {
	ProviderID           int
	ProviderType         string
	ProviderIPs          []string
	ProviderState        string
	ProviderStatus       string
	ProviderErrorMessage string
}
