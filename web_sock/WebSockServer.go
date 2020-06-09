package web_sock

import (
	//"context"

	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// настройки сервера WS
type WebSockServerSettings struct {
	NeedWork        bool     `json:"LoggerNeedWork"` // необходимость работы
	LocalPort       int      `json:"LoggerPort"`     // сетевой порт
	PermitClientIps []string // список размерешшых клиентских адресов (если пустой, то всем можно подключаться)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// данные и указатель на сокет
type WsPackage struct {
	Data []byte          // данные (принятые | для отправки)
	Sock *websocket.Conn // указатель на сокет (от которого приняли | которому необходимо передать)
}

// Контроллер, отвечающий за прием/отправку сообщений по WebSocket
type WebSockServer struct {
	sync.Mutex
	//http.Server
	ErrorChan       chan error                 // канал для передачи ошибок в работе сокета (если успешно подключились, передаем nil)
	InfoChan        chan string                // канал для передачи сообщений о работе сервера
	ReceiveDataChan chan WsPackage             // канал для отправки считанных данных
	SendDataChan    chan WsPackage             // канал для приема дынных, которые необходимо отправить клиентам
	SettingsChan    chan WebSockServerSettings // канал приема новых настроек контроллера

	curSetts WebSockServerSettings // текущие настройки контроллера

	server *http.Server // Web сервер

	clients map[*websocket.Conn]bool // подключенные клиенты. ключ- сокет, значение - успешное подключение
}

// конструктор
func NewWebSockServer() *WebSockServer {
	return &WebSockServer{
		ErrorChan:       make(chan error, 100),
		InfoChan:        make(chan string, 10),
		ReceiveDataChan: make(chan WsPackage, 1024),
		SendDataChan:    make(chan WsPackage, 1024),
		SettingsChan:    make(chan WebSockServerSettings),
		clients:         make(map[*websocket.Conn]bool),
	}
}

// запуск работы контроллера
func (nc *WebSockServer) Work(urlPath string) {
	router := mux.NewRouter()
	router.HandleFunc(urlPath, nc.handleMain)

FORLBL:
	for {
		select {
		//
		case newSettings := <-nc.SettingsChan:
			if !reflect.DeepEqual(nc.curSetts, newSettings) {
				nc.curSetts = newSettings

				if nc.server != nil {
					nc.stopNetServer()
				}

				nc.server = &http.Server{
					Handler:      router,
					Addr:         ":" + strconv.Itoa(nc.curSetts.LocalPort),
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
				}
				go func() {
					// отправляем nil ошибку (типа запускаем web сервер)
					nc.ErrorChan <- nil
					if err := nc.server.ListenAndServe(); err != nil {
						nc.ErrorChan <- fmt.Errorf("Ошибка запуска WS сервера. Ошибка: <%s>", err.Error())
					}
				}()
			}

		// получены данные для клиента (на отправку)
		case curPkgToWrite := <-nc.SendDataChan:
			nc.Lock()
			if curPkgToWrite.Sock != nil {
				//if _, ok := nc.clientSockets[curPkgToWrite.Sock]; ok {
				if _, ok := nc.clients[curPkgToWrite.Sock]; ok {
					//curPkgToWrite.Sock.SendChan <- curPkgToWrite.Data
					if err := curPkgToWrite.Sock.WriteMessage(websocket.TextMessage, curPkgToWrite.Data); err != nil {
						nc.ErrorChan <- fmt.Errorf("Ошибка отправки данных клиенту с адресом; <%s>. Ошибка: <%s> .", curPkgToWrite.Sock.RemoteAddr().String(), err.Error())
						curPkgToWrite.Sock.Close()
						delete(nc.clients, curPkgToWrite.Sock)
						continue FORLBL
					}
				} else {
					nc.ErrorChan <- fmt.Errorf("Не найден клиент для отправки. Адрес клиента: <%s>.", curPkgToWrite.Sock.RemoteAddr().String())
				}
			} else {

			SOCKLBL:
				for sock := range nc.clients {
					//sock.SendChan <- curPkgToWrite.Data
					if err := sock.WriteMessage(websocket.TextMessage, curPkgToWrite.Data); err != nil {
						nc.ErrorChan <- fmt.Errorf("Ошибка отправки данных клиенту с адресом; <%s>. Ошибка: <%s> .", sock.RemoteAddr().String(), err.Error())
						sock.Close()
						delete(nc.clients, sock)
						continue SOCKLBL
					}
				}
			}
			nc.Unlock()
		}
	}
}

func (nc *WebSockServer) handleMain(w http.ResponseWriter, r *http.Request) {
	// апгрейд соединения
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Ошибка апгрейда WS", err)
	} else {
		permitConn := false
		if len(nc.curSetts.PermitClientIps) > 0 {
			for _, it := range nc.curSetts.PermitClientIps {
				fmt.Printf("'" + it + "'-'" + client.RemoteAddr().String() + "'")
				if strings.HasPrefix(client.RemoteAddr().String(), it) {
					permitConn = true
					break
				}
			}
		} else {
			permitConn = true
		}

		if permitConn {
			nc.clients[client] = true
			nc.InfoChan <- fmt.Sprint("Подключен клиент с адресом: <%s>.", client.RemoteAddr().String())

			go func(curSock *websocket.Conn) {
				for {
					if _, data, err := curSock.ReadMessage(); err != nil {
						nc.ErrorChan <- fmt.Errorf("Ошибка чтения данных от клиента с адресом; <%s>. Ошибка: <%s>.", curSock.RemoteAddr().String(), err.Error())
						curSock.Close()
						delete(nc.clients, curSock)
						return
					} else {
						nc.ReceiveDataChan <- WsPackage{Data: data, Sock: curSock}
					}
				}
			}(client)
		} else {
			nc.InfoChan <- fmt.Sprint("Отклонено подключение клиента с адресом: <%s>.", client.RemoteAddr().String())
			if err := client.Close(); err != nil {
				nc.ErrorChan <- fmt.Errorf("Ошибка закрытия клиентского подключения из-за недопустимого адреса клиента. Ошибка: <%s>.", err.Error())
			} else {
				nc.ErrorChan <- fmt.Errorf("Закрыто клиентское подключение, недопустимый адрес клиента.")
			}
			return
		}
	}
}

func (nc *WebSockServer) stopNetServer() {
	for sock := range nc.clients {
		sock.Close()
		delete(nc.clients, sock)
	}

	if err := nc.server.Close(); err != nil {
		nc.ErrorChan <- fmt.Errorf("Ошибка закрытия WS сервера.", err.Error())
	}
	nc.server = nil
}

// IsIPConnected предоставляет сведения о состоянии подключения клиента с указанным IP адресом
func (nc *WebSockServer) IsIPConnected(clientIP string) bool {
	for key := range nc.clients {
		if strings.HasPrefix(key.RemoteAddr().String(), clientIP) {
			return true
		}
	}
	return false
}
