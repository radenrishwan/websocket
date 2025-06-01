package websocket

import (
	"crypto/sha1"
	"encoding/base64"
)

func generateWebsocketKey(key string) string {
	sha := sha1.New()
	sha.Write([]byte(key))
	sha.Write([]byte(MAGIC_KEY))

	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}
