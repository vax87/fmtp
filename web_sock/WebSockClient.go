package web_sock

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// настройки WS
type WebSockClientSettings struct {
	ServerAddress     string // сетевой адрес сервера WS
	ServerPort        int    // сетевой порт сервера WS
	UrlPath           string // path
	ReconnectInterval int    // интервал переподключения к серверу в случае ошибки
	// если = -1, значит не недо переподключаться (серверный сокет)
}

const (
	WriteWait         = 10 * time.Second    // Time allowed to write a message to the peer.
	MaxMessageSize    = 8192                // Maximum message size allowed from peer.
	PongWait          = 6 * time.Second     // Time allowed to read the next pong message from the peer.
	PingPeriod        = (PongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
	CloseGracePeriod  = 10 * time.Second    // Time to wait before force close on connection.
	ReconnectDuration = 5 * time.Second     // интервал переподключения к серверу
)

// обертка Web Socket
type WebSockClient struct {
	sync.Mutex

	ErrorChan chan error // канал для передачи ошибок в работе сокета (если успешно подключились, передаем nil)

	ReceiveDataChan chan []byte // канал для приема данных от контроллера каналов
	SendDataChan    chan []byte // канал для отправки данных контроллеру каналов

	cancelWorkChan   chan struct{}         // канал для сигнала отмены отправки/приема данных
	connectErrorChan chan error            // канал для передачи ошибки при подключении
	workErrorChan    chan error            // канал для передачи ошибки во время работы
	ws               *websocket.Conn       // клиентское подключение по WS
	wsSettings       WebSockClientSettings // текущие настройки WS

	lastConnError error // последняя возникшая ошибка подключения
	WsConnected   bool  // признак работоспособности (подключен к серверу)
}

// * при возникновении ошибки в чтении или записи, рутина, в которой возникла ошибка отправляет в канал
// errorChan ошибку и завершается, вторая рутина вычитывает из канала cancelWorkChan и тоже завершается
// поэтому чтение из cancelWorkChan в обеих рутинах, а пишется в него один раз в stopClient

// конструктор
func NewWebSockClient() *WebSockClient {
	return &WebSockClient{
		ErrorChan:        make(chan error, 100),
		ReceiveDataChan:  make(chan []byte, 1024),
		SendDataChan:     make(chan []byte, 1024),
		cancelWorkChan:   make(chan struct{}),
		connectErrorChan: make(chan error),
		workErrorChan:    make(chan error),
		lastConnError:    errors.New(""),
		WsConnected:      false,
	}
}

func (wsc *WebSockClient) Work(wsSetts WebSockClientSettings) {
	if wsc.wsSettings != wsSetts {
		wsc.wsSettings = wsSetts

		go wsc.startClient()

		for {
			select {
			// ошибка подключения к серверу
			case <-wsc.connectErrorChan:
				time.AfterFunc(ReconnectDuration, wsc.startClient)

			// ошибка чтения/записи
			case err := <-wsc.workErrorChan:
				wsc.WsConnected = false
				wsc.ErrorChan <- err
				wsc.stopClient()
				time.AfterFunc(ReconnectDuration, wsc.startClient)
			}
		}
	}
}

// предоставить признак работоспособности (подключен к серверу)
func (wsc *WebSockClient) IsConnected() bool {
	return wsc.WsConnected
}

// стартовать подключение к серверу
func (wsc *WebSockClient) startClient() {
	wsAddr := fmt.Sprintf("%s:%d", wsc.wsSettings.ServerAddress, wsc.wsSettings.ServerPort)
	u := url.URL{Scheme: "ws", Host: wsAddr, Path: wsc.wsSettings.UrlPath}
	var err error
	if wsc.ws, _, err = websocket.DefaultDialer.Dial(u.String(), nil); err != nil {
		wsc.connectErrorChan <- err
		if err.Error() != wsc.lastConnError.Error() {
			wsc.lastConnError = err
			wsc.ErrorChan <- err
		}
		return
	} else {
		wsc.WsConnected = true
		wsc.ErrorChan <- nil
		go wsc.receiveLoop()
		go wsc.sendLoop()
	}
}

// остановить соединение с сервером
func (wsc *WebSockClient) stopClient() {
	wsc.Lock()
	wsc.cancelWorkChan <- struct{}{}
	wsc.Unlock()

	wsc.ws.Close()
}

// обработчик получения данных
func (wsc *WebSockClient) receiveLoop() {
	for {
		select {
		// получена отмена отправки/приема данных
		case <-wsc.cancelWorkChan:
			return

		default:
			_, data, err := wsc.ws.ReadMessage()
			if err != nil {
				wsc.workErrorChan <- err
				return
			} else {
				wsc.ReceiveDataChan <- data
			}
		}
	}
}

// обработчик отправки данных
func (wsc *WebSockClient) sendLoop() {
	for {
		select {
		// получена отмена отправки/приема данных
		case <-wsc.cancelWorkChan:
			return
		// получены данные для отправки кленту
		case curDataToSend := <-wsc.SendDataChan:
			if err := wsc.ws.WriteMessage(websocket.TextMessage, curDataToSend); err != nil {
				wsc.workErrorChan <- err
				return
			}
		}
	}
}
