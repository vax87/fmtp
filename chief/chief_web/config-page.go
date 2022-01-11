package chief_web

import (
	"html/template"
	"sync"

	"fdps/fmtp/chief_configurator/configurator_urls"
	"fdps/go_utils/logger"
)

type ConfigPage struct {
	sync.RWMutex
	editTempl *template.Template
	Title     string

	UrlConfig      configurator_urls.ConfiguratorUrls
	WriteStateToDb bool
}

func (cp *ConfigPage) initialize(title string) {
	cp.Lock()
	defer cp.Unlock()

	var err error
	if cp.editTempl, err = template.New("Config").Parse(ConfigTemplate); err != nil {
		logger.PrintfErr("EditConfig template Parse ERROR: %v", err)
		return
	}
	cp.Title = title
}

var ConfigTemplate = `
<title>{{.Title}}</title>
<h1></h1>

<form action="/saveConfig" method="POST">

<table width="100%" cellspacing="0" cellpadding="4">
	<tr>
		<td colspan="2">Настройки подключения к БД</td>
	</tr>
	{{with .UrlConfig}}
		<tr>
			<td align="left">URL для запроса настроек:</td>
			<td><input name="SettingsURLStr" type="text" size="100" value={{printf "%s" .SettingsURLStr}}></td>
		</tr>
		<tr>
			<td align="left">URL для отправки состояния:</td>
			<td><input name="HeartbeatURLStr" type="text" size="100" value={{printf "%s" .HeartbeatURLStr}}></td>			
		</tr>
		
		<tr>
		<tr>
			<td colspan="2"></td>
		</tr>
		<td>
			<input type="checkbox" name="WriteStateToDb" value="check" {{if .WriteStateToDb}} checked {{end}} />Сохранять состояние каналов в БД:<Br>
		</td>
	</tr>
	{{end}}	
	<tr> 
    	<td colspan="2"><input type="submit" value="Применить"></td>
	</tr>	
</table>
</form>
`
