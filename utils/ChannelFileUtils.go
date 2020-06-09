package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	//"strings"
)

var versionDir = AppPath() + "/versions"
var runningChannelDir = AppPath() + "/runningChannels"
var binExtension = ".exe"
var channelBin = "fmtp_channel" + binExtension

// GetChannelVersions предоставляет список версий приложения FMTP канала
func GetChannelVersions() ([]string, error) {
	var retValue []string

	dirs, dirErr := ioutil.ReadDir(versionDir)
	if dirErr != nil {
		return []string{}, fmt.Errorf("Ошибка чтения директории с версиями исполняемых файлов FMTP канала. Директория <%s>. Ошибка: <%s>",
			versionDir, dirErr.Error())
	} else {
		for _, dir := range dirs {
			if dir.IsDir() {
				curVersionDir := versionDir + "/" + dir.Name()
				files, fileErr := ioutil.ReadDir(curVersionDir)
				if fileErr != nil {
					return retValue, fmt.Errorf("Ошибка чтения директории с исполняемым файлом FMTP канала. Директория <%s>. Ошибка: <%s>",
						curVersionDir, fileErr.Error())
				} else {
					// проверка, что содержится исполняемого файл канала
					for _, file := range files {
						fmt.Println("file ", file.Name())
						if !file.IsDir() && file.Name() == channelBin {
							retValue = append(retValue, dir.Name())
							break
						}
					}
				}
			}
		}
	}

	if len(retValue) == 0 {
		return retValue, fmt.Errorf("Не найдена ни одна версия исполняемого файла FMTP канала. Директория: <%s>", (AppPath() + "/versions"))
	}

	return retValue, nil
}

// CopyChannelBinary копирование приложение FMTP канала
func CopyChannelBinary(vers string, id int, localAtc string, remoteAtc string) (binFilePath string, err error) {
	if chDirErr := os.Chdir(runningChannelDir); chDirErr != nil {
		if mkdirErr := os.MkdirAll(runningChannelDir, os.ModePerm); mkdirErr != nil {
			return "", fmt.Errorf("Невозможно создать рабочую директорию для запуска каналов FMTP. Ошибка: <%s>", mkdirErr.Error())
		}
	}

	srcDir := versionDir + "/" + vers
	if chDirErr := os.Chdir(runningChannelDir); chDirErr != nil {
		return "", fmt.Errorf("Ошибка копирования исполняемого файла FMTP канала версии <%s>. Версия не найдена. Ошибка: <%s>", vers, chDirErr.Error())
	}
	srcFile := srcDir + "/" + channelBin

	input, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return "", fmt.Errorf("Ошибка копирования файла <%s> в рабочую директорию FMTP каналов. Ошибка: <%s>", srcFile, err.Error())
	}

	dstFile := runningChannelDir + "/" + fmt.Sprintf("fmtp_channel_%d_%s_%s", id, localAtc, remoteAtc) + binExtension

	if err := ioutil.WriteFile(dstFile, input, os.ModePerm); err != nil {
		return "", fmt.Errorf("Ошибка копирования файла <%s> в рабочую директорию FMTP каналов. Ошибка: <%s>", dstFile, err.Error())
	}
	return dstFile, nil
}

// RemoveChannelBinary копирует исполняемый файл приложения FMTP канала
func RemoveChannelBinary(binFilePath string) error {
	if err := os.Remove(binFilePath); err != nil {
		return fmt.Errorf("Ошибка при удалении исполняемого файла FMTP канала из рабочей директории. Исполняемый файл: <%s>. Ошибка: <%s>", binFilePath, err.Error())
	}
	return nil
}
