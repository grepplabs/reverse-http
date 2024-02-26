package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestTraceLevel(t *testing.T) {
	InitInstance(LogConfig{
		Level: "trace",
	})
	log := GetInstance()
	ctx := context.Background()
	log.Log(ctx, LevelTrace, "Trace message")
	log.Log(ctx, slog.LevelInfo, "Info message")
}

func TestLoggerIntf(t *testing.T) {
	InitInstance(LogConfig{
		Level: "trace",
	})
	log := GetInstance()

	log.Trace("Hello logger - trace")
	log.Trace("Hello logger - trace", slog.String("tag", "value"))
	log.Error("Hello logger - error")

	log.Tracef("Tracef %s -> %s", "from", "to")
	log.Debugf("Debugf  %s -> %s", "from", "to")
	log.Infof("Infof  %s -> %s", "from", "to")
	log.Warnf("Warnf %s -> %s", "from", "to")
	log.Errorf("Errorf %s -> %s", "from", "to")

	log.WithFields(nil).Info("Hello fields")
	log.WithFields(map[string]any{}).Info("Hello fields")
	log.WithFields(map[string]any{"tag1": "a"}).Info("Hello fields")
	log.WithFields(map[string]any{"tag1": "a", "tag2": 1}).Info("Hello fields")
}
