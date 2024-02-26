package none

import (
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/store"
)

type client struct {
	logger *logger.Logger
}

func NewClient() store.Client {
	return &client{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "store-none"}),
	}
}

func (c *client) Get(key string) (string, error) {
	c.logger.Debugf("get %s", key)
	return "", nil
}

func (c *client) Set(key, value string) error {
	c.logger.Debugf("set %s to %s", key, value)
	return nil
}

func (c *client) Delete(key string, value string) error {
	c.logger.Debugf("delete %s value %s", key, value)
	return nil
}

func (c *client) Close() {
}
