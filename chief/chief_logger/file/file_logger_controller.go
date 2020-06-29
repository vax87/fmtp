package file

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"

	"fdps/fmtp/chief/chief_logger/common"
)

var fileLogger lumberjack.Logger

func init() {
	// директория, в которую будут складываться файлы с логами
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir += "/logs"

	// создание директории, в которую будут складываться файлы с логами
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println(err)
	}

	// инициализация логера
	fileLogger = lumberjack.Logger{
		Filename:   dir + filepath.FromSlash("/fmtp_logs.log"),
		MaxSize:    1,    // megabytes
		MaxBackups: 1024, // count files
		MaxAge:     30,   //days
		Compress:   true, // disabled by default
	}

	// выставляем микросекунды в записи времени логов
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	// перенаправление логов
	log.SetOutput(&fileLogger)
}

// контроллер, отвечающий за запись логов в файлы
type FileLoggerController struct {
	SettingsChan chan FileLoggerSettings // канал, принимающий настройки контроллера
	MessChan     chan common.LogMessage  // канал, принимающий сообщения

	currentSettings FileLoggerSettings // текщие настройки контроллера
}

// CreateFileLoggerController - конструктор
func CreateFileLoggerController() *FileLoggerController {
	return &FileLoggerController{
		MessChan:     make(chan common.LogMessage, 1024),
		SettingsChan: make(chan FileLoggerSettings),
	}
}

// работа контроллера
func (flc *FileLoggerController) Run() {
	for {
		select {
		case newSettings := <-flc.SettingsChan:
			if flc.currentSettings != newSettings {
				flc.currentSettings = newSettings

				fileLogger.MaxSize = int(math.Ceil(float64(flc.currentSettings.LogFileSizeKb) / 1024.0))
				fileLogger.MaxBackups = int(math.Ceil(
					float64(flc.currentSettings.LogFolderSizeGb) * 1024.0 * 1024.0 /
						float64(flc.currentSettings.LogFileSizeKb)))
				fileLogger.MaxAge = flc.currentSettings.LogStoreDays

				fileLogger.Rotate()
				log.Printf("Получены новые настройки логера в файлы: %+v", flc.currentSettings)
			}
		case newMessage := <-flc.MessChan:
			log.Printf("Получено сообщение: %+v", newMessage)
		}
	}
}
