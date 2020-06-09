package web_sock

//import (
//	"fmt"
//	"sync"
//	"time"
//
//	"github.com/gorilla/websocket"
//)
//
//type WebSockServerSocket struct {
//	sync.Mutex
//
//	ErrorChan chan error // канал для передачи ошибок в работе сокета
//
//	ReceiveChan chan []byte // канал для отправки данных
//	SendChan    chan []byte // канал для приема данных
//
//	cancelWorkChan chan struct{} // канал для сигнала отмены отправки/приема данных
//
//	workErrorChan chan error      // канал для передачи ошибки во время работы
//	sock          *websocket.Conn // клиентское подключение по WS
//}
//
//// конструктор
//func newNetClient(ws *websocket.Conn) *WebSockServerSocket {
//	return &WebSockServerSocket{
//		ErrorChan:      make(chan error),
//		ReceiveChan:    make(chan []byte, 1024),
//		SendChan:       make(chan []byte, 1024),
//		cancelWorkChan: make(chan struct{}),
//		workErrorChan:  make(chan error),
//		sock:           ws,
//	}
//}
//
////
//func (ws *WebSockServerSocket) startClient() {
//	go ws.receiveLoop()
//	go ws.sendLoop()
//
//	for {
//		select {
//		// ошибка чтения/записи
//		case err := <-ws.workErrorChan:
//			ws.stopClient()
//			ws.ErrorChan <- err
//		}
//	}
//}
//
////
//func (ws *WebSockServerSocket) stopClient() {
//	ws.Lock()
//	ws.cancelWorkChan <- struct{}{}
//	ws.Unlock()
//
//	ws.sock.Close()
//}
//
//// обработчик получения данных
//func (ws *WebSockServerSocket) receiveLoop() {
//	for {
//		select {
//		// получена отмена отправки/приема данных
//		case <-ws.cancelWorkChan:
//			return
//
//		default:
//			if _, data, err := ws.sock.ReadMessage(); err != nil {
//				fmt.Println(time.Now().String(), " WS Server Sock Receive ERROR ", err.Error())
//				ws.workErrorChan <- err
//				return
//			} else {
//				ws.ReceiveChan <- data
//			}
//		}
//	}
//	//
//	//for {
//	//	if !nclnt.needWork {
//	//		fmt.Println(time.Now().String(), "Cancel read data")
//	//		return
//	//	}
//	//	_, data, err := nclnt.clientWs.ReadMessage()
//	//	if err != nil {
//	//		fmt.Println("Ошибка чтения сообщения ", err)
//	//		nclnt.stopClient()
//	//
//	//	} else {
//	//		nclnt.receiveDataChan <- data
//	//	}
//	//}
//}
//
//// обработчик отправки данных
//func (ws *WebSockServerSocket) sendLoop() {
//	for {
//		select {
//		// получена отмена отправки/приема данных
//		case <-ws.cancelWorkChan:
//			return
//		// получены данные для отправки кленту
//		case curDataToSend := <-ws.SendChan:
//			if err := ws.sock.WriteMessage(websocket.TextMessage, curDataToSend); err != nil {
//				fmt.Println(time.Now().String(), " WS Server Sock Send ERROR ", err.Error())
//				ws.workErrorChan <- err
//				return
//			}
//		}
//	}
//
//	//for {
//	//	select {
//	//	// получены данные для отправки кленту
//	//	case curDataToSend := <-nclnt.dataToSendChan:
//	//		if err := nclnt.clientWs.WriteMessage(websocket.TextMessage, curDataToSend); err != nil {
//	//			fmt.Println("Ошибка отправки сообщения ", err)
//	//			nclnt.stopClient()
//	//		}
//	//
//	//	// получена отмена отправки данных
//	//	case <-nclnt.cancelSendChan:
//	//		fmt.Println(time.Now().String(), "Cancel write data")
//	//		return
//	//	}
//	//}
//}
