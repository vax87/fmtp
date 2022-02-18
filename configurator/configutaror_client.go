package configurator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fmtp/configurator/configurator_urls"
	"io/ioutil"
	"net/http"
	"strings"
)

type httpResult struct {
	result json.RawMessage
	err    error
}

// клиент контроллера для подключения к конфигуратору
type ConfiguratorClient struct {
	configUrls     configurator_urls.ConfiguratorUrls
	postResultChan chan httpResult
}

func (c *ConfiguratorClient) getUrls() {
	c.configUrls.ReadFromFile()
}

func (c *ConfiguratorClient) setUrls() {
	c.configUrls.SaveToFile()
}

// отправка POST сообщения конфигуратору и возврат ответа
func (c *ConfiguratorClient) postToConfigurator(url string, msg interface{}) {
	jsonValue, _ := json.Marshal(msg)
	//logger.PrintfWarn("POST: %s", string(jsonValue))
	resp, postErr := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if postErr == nil {
		defer resp.Body.Close()
		if strings.Contains(resp.Status, "200") {
			if body, readErr := ioutil.ReadAll(resp.Body); readErr == nil {
				if bytes.Contains(body, []byte("error")) { //пишем в логи только ошибки

					c.postResultChan <- httpResult{err: fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))}
				}
				if ind := strings.Index(string(body), "{"); ind >= 0 {
					c.postResultChan <- httpResult{result: body[ind:], err: nil}
				} else {
					c.postResultChan <- httpResult{err: fmt.Errorf("Не валидное тело http пакета. Тело пакета: %s", string(body))}
				}
			} else {
				c.postResultChan <- httpResult{err: fmt.Errorf("Ошибка чтения ответа http запроса. Тело пакета: %s. Ошибка: %s", string(body), readErr.Error())}
			}
		} else {
			c.postResultChan <- httpResult{err: fmt.Errorf("HTPP запрос выполнен с ошибкой. Запрос: %s. URL: %s. Статус ответа: %s",
				jsonValue, url, resp.Status)}
		}
	} else {
		c.postResultChan <- httpResult{err: fmt.Errorf("Ошибка выполнения HTPP запроса. Запрос: %s. URL: %s. Ошибка: %s",
			jsonValue, url, postErr.Error())}
	}
}
