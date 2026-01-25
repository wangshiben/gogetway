package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var LoggerFile *os.File
var Logger *log.Logger

const debug = false

func InitLogger() {
	if debug {
		now := time.Now().Format("2006_01_02_15_04_05")
		fmt.Println(now)
		fileName := fmt.Sprintf("%s.log", now)
		create, err := os.Create(fileName)
		if err != nil {
			return
		}
		LoggerFile = create
		Logger = log.New(LoggerFile, "Info", log.LstdFlags)
	} else {
		discard := io.Discard
		Logger = log.New(discard, "", 0)
	}

}
func LogInfo(format string) {
	Logger.Println(format)
}
