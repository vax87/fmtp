package chief_web

import (
	"html/template"
	"sync"

	"fdps/utils/logger"
)

type ConfigPage struct {
	sync.RWMutex
	editTempl *template.Template
	Title     string

	//Setts settings.Settings
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
	{{with .Setts}}
		{{with .DbSetts}}
		<tr>
			<td align="left">Хост:</td>
			<td><input name="DbHostname" type="text" size="40" value={{printf "%s" .Hostname}}></td>
		</tr>
		<tr>
			<td align="left">Порт:</td>
			<td><input name="DbPort" type="number" min="1024" max="65535" size="40" value={{printf "%d" .Port}}></td>
		</tr>
		<tr>
			<td align="left">Имя сервиса:</td>
			<td><input name="DbServiceName" type="text" size="40" value={{printf "%s" .ServiceName}}></td>
		</tr>
		<tr>
			<td align="left">Пользователь:</td>
			<td><input name="DbUser" type="text" size="40" value={{printf "%s" .UserName}}></td>
		</tr>
		<tr>
			<td align="left">Пароль:</td>
			<td><input name="DbPassword" type="text" size="40" value={{printf "%s" .Password}}></td>
		</tr>
		{{end}}
	<tr>
		<td colspan="2"></td>
	</tr>
	<tr>
		<td colspan="2"></td>
	</tr>
	<tr>
		<td align="left">ICAO код FIR:</td>
		<td><input name="FirCode" type="text" size="40" value={{printf "%s" .FirCode}}></td>
	</tr>
	<tr>
		<td>
		<input type="checkbox" name="NeedUpdateANI" value="check" />Обновить АНИ<Br>
		</td>
	</tr>
	<tr>
		<td colspan="2"></td>
	</tr>
	<tr>
		<td colspan="2"></td>
	</tr>
		{{with .WsSetts}}
		<tr>
			<td colspan="2">Настройки подключения клиентов</td>
		</tr>
		
		<tr>
			<td align="left">Порт:</td>
			<td><input name="WsPort" type="number" min="1024" max="65535" size="40" value={{printf "%d" .Port}}></td>
		</tr>
		{{end}}	
	{{end}}	
	<tr> 
    	<td colspan="2"><input type="submit" value="Применить"></td>
	</tr>	
</table>
</form>
`
