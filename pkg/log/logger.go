package log

import (
  "fmt"
  "os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func GetLogger(defaultLogLevel zerolog.Level) zerolog.Logger {
	zerolog.SetGlobalLevel(defaultLogLevel)

	logWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	logWriter.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	logWriter.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	logWriter.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	logWriter.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}

	return zerolog.New(logWriter).With().Timestamp().Logger()
}
