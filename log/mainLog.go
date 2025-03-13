package log

import (
	"fmt"
	"log"
	"os"
)

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
)

type LogLevel string

const (
	SUCS    LogLevel = "SUCCESS"
	NRML    LogLevel = "NORMAL"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

func LogMessage(level LogLevel, message interface{}) {
	log.SetOutput(os.Stdout)
	var color string

	switch level {
	case SUCS:
		color = Green
	case NRML:
		color = ""
	case INFO:
		color = Blue
	case WARNING:
		color = Yellow
	case ERROR:
		color = Red
	default:
		color = Reset
	}

	formattedMessage := fmt.Sprintf("%s[%s]%s %v", color, level, Reset, message)

	log.Println(formattedMessage)
}
