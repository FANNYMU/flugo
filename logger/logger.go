package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"flugo.com/config"
)

type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	TRACE: "TRACE",
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var levelColors = map[Level]string{
	TRACE: "\033[36m", // Cyan
	DEBUG: "\033[34m", // Blue
	INFO:  "\033[32m", // Green
	WARN:  "\033[33m", // Yellow
	ERROR: "\033[31m", // Red
	FATAL: "\033[35m", // Magenta
}

const colorReset = "\033[0m"

type Logger struct {
	level  Level
	format string
	writer io.Writer
	prefix string
}

var DefaultLogger *Logger

func Init(cfg *config.LoggerConfig) {
	level := parseLevel(cfg.Level)

	var writer io.Writer = os.Stdout
	if cfg.OutputFile != "" {
		file, err := os.OpenFile(cfg.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			writer = io.MultiWriter(os.Stdout, file)
		}
	}

	DefaultLogger = &Logger{
		level:  level,
		format: cfg.Format,
		writer: writer,
		prefix: "",
	}
}

func parseLevel(levelStr string) Level {
	switch strings.ToUpper(levelStr) {
	case "TRACE":
		return TRACE
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	_, file, line, _ := runtime.Caller(2)
	filename := file[strings.LastIndex(file, "/")+1:]

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	var logLine string
	if l.format == "json" {
		logLine = fmt.Sprintf(`{"timestamp":"%s","level":"%s","file":"%s:%d","message":"%s"}`,
			timestamp, levelName, filename, line, message)
	} else {
		color := levelColors[level]
		if l.writer == os.Stdout {
			logLine = fmt.Sprintf("%s[%s]%s %s %s:%d - %s",
				color, levelName, colorReset, timestamp, filename, line, message)
		} else {
			logLine = fmt.Sprintf("[%s] %s %s:%d - %s",
				levelName, timestamp, filename, line, message)
		}
	}

	fmt.Fprintln(l.writer, logLine)

	if level == FATAL {
		os.Exit(1)
	}
}

func (l *Logger) Trace(format string, args ...interface{}) {
	l.log(TRACE, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		level:  l.level,
		format: l.format,
		writer: l.writer,
		prefix: prefix,
	}
}

func Trace(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Trace(format, args...)
	} else {
		log.Printf("[TRACE] "+format, args...)
	}
}

func Debug(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(format, args...)
	} else {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Info(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Error(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

func Fatal(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal(format, args...)
	} else {
		log.Fatalf("[FATAL] "+format, args...)
	}
}
