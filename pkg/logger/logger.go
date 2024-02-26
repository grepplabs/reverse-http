package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

var LevelNames = map[slog.Leveler]string{
	LevelTrace: "TRACE",
	LevelFatal: "FATAL",
}
var LevelMap = invertMap(LevelNames)

const (
	EnvLogFormat = "LOG_FORMAT"
	EnvLogLevel  = "LOG_LEVEL"
)

const (
	LogFormatJson = "json"
	LogFormatText = "text"
)

type LogConfig struct {
	Format string `env:"LOG_FORMAT" enum:"json, text" default:"text" help:"Log format. One of: [json, text]"`
	Level  string `env:"LOG_LEVEL" enum:"trace, debug, info, warn, error" default:"info" help:"Log level. One of: [trace, debug, info, warn, error]"`
}

var (
	instance *Logger
	once     sync.Once
)

func InitInstance(config LogConfig) {
	once.Do(func() {
		instance = NewLogger(config)
		slog.SetDefault(instance.Logger)
	})
}

func GetInstance() *Logger {
	once.Do(func() {
		if instance == nil {
			instance = NewDefaultLogger()
		}
	})
	return instance
}

func NewLogger(config LogConfig) *Logger {
	var level slog.Level

	if l, ok := LevelMap[strings.ToUpper(config.Level)]; ok {
		level = l
	} else {
		if err := level.UnmarshalText([]byte(config.Level)); err != nil {
			level = slog.LevelInfo
		}
	}
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if l, ok := a.Value.Any().(slog.Level); ok {
					if levelLabel, levelExists := LevelNames[l]; levelExists {
						a.Value = slog.StringValue(levelLabel)
					}
				}
			}
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			}
			return a
		},
	}
	var handler slog.Handler
	switch config.Format {
	case LogFormatJson:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	case LogFormatText:
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	return &Logger{
		slog.New(handler),
	}
}

func NewDefaultLogger() *Logger {
	config := LogConfig{
		Format: getEnv(EnvLogFormat, LogFormatText),
		Level:  getEnv(EnvLogLevel, slog.LevelInfo.String()),
	}
	return NewLogger(config)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func invertMap(m map[slog.Leveler]string) map[string]slog.Level {
	inverted := make(map[string]slog.Level)
	for k, v := range m {
		inverted[v] = k.Level()
	}
	return inverted
}

type Logger struct {
	*slog.Logger
}

func (l *Logger) Trace(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelTrace, msg, args...)
}

func (l *Logger) Tracef(format string, args ...any) {
	l.Logger.Log(context.Background(), LevelTrace, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.Logger.Log(context.Background(), LevelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *Logger) Infof(format string, args ...any) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Debugf(format string, args ...any) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

func (l *Logger) IsLevelEnabled(level slog.Level) bool {
	return l.Enabled(context.Background(), level)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

func (l *Logger) WithFields(fields map[string]any) *Logger {
	if len(fields) == 0 {
		return l
	}
	args := make([]any, 0, len(fields))
	for k, v := range fields {
		args = append(args, slog.Any(k, v))
	}
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}
