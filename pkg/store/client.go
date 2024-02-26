package store

type Client interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key, value string) error
	Close()
}
