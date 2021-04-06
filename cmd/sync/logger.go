package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// logger is the configured zerolog Logger instance.
var logger zerolog.Logger

func init() {
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

	logger = zerolog.New(logWriter).With().Timestamp().Logger()
}
