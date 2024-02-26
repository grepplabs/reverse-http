package memcached

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grepplabs/reverse-http/config"
	"github.com/stretchr/testify/require"
)

func TestMemcached(t *testing.T) {
	t.Skip("Integration test")

	key := uuid.NewString()

	client := NewClient(config.MemcachedConfig{
		Address: "localhost:11211",
		Timeout: 1 * time.Second,
	})
	defer client.Close()

	for i := 0; i < 3; i++ {
		v, err := client.Get(key)
		require.NoError(t, err)
		require.Equal(t, "", v)
	}
	for i := 0; i < 3; i++ {
		v := strconv.Itoa(i)
		err := client.Set(key, v)
		require.NoError(t, err)

		value, err := client.Get(key)
		require.NoError(t, err)
		require.Equal(t, v, value)
	}
	for i := 0; i < 3; i++ {
		err := client.Delete(key, "2")
		require.NoError(t, err)

		v, err := client.Get(key)
		require.NoError(t, err)
		require.Equal(t, "", v)

	}
}
