package chief_web

import (
	"fmtp/channel/channel_state"
	"fmtp/chief/chief_state"
	"html/template"
	"sync"
)

// ChiefPage страница контроллера
type ChiefPage struct {
	sync.RWMutex
	templ *template.Template
	Title string

	ChannelStates      []channel_state.ChannelState
	OldiProviderStates []chief_state.ProviderState
	AodbProviderStates []chief_state.ProviderState
}

func (m *ChiefPage) initialize(title string) {
	m.Lock()
	defer m.Unlock()

	var err error
	if m.templ, err = template.New("ChiefTemplate").Parse(ChiefPageTemplate); err != nil {
		return
	}

	m.Title = title
}

var ChiefPageTemplate = `{{define "ChiefTemplate"}}
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
		<script>
		//    window.onload = function() {
		//        setTimeout(function () {
		//            location.reload()
		//        }, 1000);
		//     };
		</script>
	</head>
	<body style="background-color:#EAECEE;">
		<font size="4" face="verdana" color="black">
		
			<table width="100%" border="1" cellspacing="0" cellpadding="4" >
				<caption style="font-weight:bold">FMTP каналы</caption>
				<tr>
					<th>ID</th>
					<th>Состояние</th>		
					<th>FMTP остояния</th>
					<th>Лок ATC</th>
					<th>Уд ATC</th>
					<th>URL</th>			
				</tr>
				{{with .ChannelStates}}
					{{range .}}
						<tr align="center" bgcolor="{{.StateColor}}">	
							<td align="left"> {{.ChannelID}} </td>	
							<td align="left"> {{.DaemonState}} </td>
							<td align="left"> {{.FmtpState}} </td>							
							<td align="left"> {{.LocalName}} </td>
							<td align="left"> {{.RemoteName}} </td>
							<td align="left"> <a href="{{.ChannelURL}}" style="display:block;">{{.ChannelURL}}</a> </td>					
						</tr>
					{{end}}
				{{end}}
			</table>

			<b>   </br>
			<b>   </br>
			<b>   </br>

			<table width="100%" border="1" cellspacing="0" cellpadding="4" >
				<caption style="font-weight:bold">Провайдеры AODB</caption>			
				<tr>
					<th>ID</th>
					<th>Состояние</th>
					<th>Список клиентов</th>			
				</tr>
				{{with .AodbProviderStates}}
					{{range .}}
						<tr align="center" bgcolor="{{.StateColor}}">	
							<td align="left"> {{.ProviderID}} </td>
							<td align="left"> {{.ProviderState}} </td>
							<td align="left"> {{.ClientAddresses}} </td>				
						</tr>
					{{end}}
				{{end}}
			</table>

			<b>   </br>
			<b>   </br>
			<b>   </br>

			<table width="100%" border="1" cellspacing="0" cellpadding="4" >
				<caption style="font-weight:bold">Провайдеры OLDI</caption>			
				<tr>
					<th>ID</th>
					<th>Состояние</th>
					<th>Список клиентов</th>		
				</tr>
				{{with .OldiProviderStates}}
					{{range .}}
						<tr align="center" bgcolor="{{.StateColor}}">	
							<td align="left"> {{.ProviderID}} </td>
							<td align="left"> {{.ProviderState}} </td>	
							<td align="left"> {{.ClientAddresses}} </td>			
						</tr>
					{{end}}
				{{end}}
			</table>
		</font>
	</body>
</html>
{{end}}
`
