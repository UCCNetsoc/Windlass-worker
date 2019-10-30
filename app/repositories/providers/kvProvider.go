package providers

type KVProvider interface {
	Put(pathPrefix string, kv map[string]interface{}) error
	Get(pathPrefix string) (map[string]interface{}, error)
}
