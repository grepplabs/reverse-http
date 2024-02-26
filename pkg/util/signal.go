package util

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/oklog/run"
)

func AddQuitSignal(group *run.Group) {
	quit := make(chan os.Signal, 1)
	group.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			logger.GetInstance().Infof("received signal %s", sig)
			return nil
		case <-quit:
			return nil
		}
	}, func(error) {
		close(quit)
	})
}
