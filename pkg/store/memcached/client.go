package memcached

import (
	"errors"
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/grepplabs/reverse-http/config"
	"github.com/grepplabs/reverse-http/pkg/logger"
	"github.com/grepplabs/reverse-http/pkg/store"
)

type client struct {
	mc     *memcache.Client
	logger *logger.Logger
}

func NewClient(conf config.MemcachedConfig) store.Client {
	mc := memcache.New(conf.Address)
	if conf.Timeout > 0 {
		mc.Timeout = conf.Timeout
	}
	return &client{
		mc:     mc,
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "memcached"}),
	}
}

func (c *client) Get(key string) (string, error) {
	value, err := c.mc.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return "", nil
		}
		return "", err
	}
	c.logger.Infof("get %s result %s", key, string(value.Value))
	return string(value.Value), nil
}

func (c *client) Set(key, value string) error {
	c.logger.Infof("set %s to %s", key, value)

	oldItem, err := c.mc.Get(key)
	if err != nil {
		if !errors.Is(err, memcache.ErrCacheMiss) {
			return err
		}
		err = c.mc.Set(&memcache.Item{Key: key, Value: []byte(value)})
		if err != nil {
			return err
		}
		setItem, err := c.mc.Get(key)
		if err != nil {
			return err
		}
		setValue := string(setItem.Value)
		if setValue != value {
			return fmt.Errorf("set and get difference")
		}
		return nil
	} else {
		oldItem.Value = []byte(value)
		err = c.mc.CompareAndSwap(oldItem)
		if err != nil {
			return err
		}
		return nil
	}
}

func (c *client) Delete(key, value string) error {
	c.logger.Infof("delete %s value %s", key, value)

	oldItem, err := c.mc.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil
		}
		return err
	}
	oldValue := string(oldItem.Value)
	if oldValue != value {
		return fmt.Errorf("delete and get difference")
	}
	err = c.mc.Delete(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil
		}
		return err
	}
	return nil
}

func (c *client) Close() {
	c.logger.Info("close client")
	_ = c.mc.Close()
}
