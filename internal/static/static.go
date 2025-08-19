package static

import (
	"net/http"
)

// BuildStaticHandler: serve static directory (đơn giản, tin cậy)
func BuildStaticHandler(dir string) http.Handler {
	return http.FileServer(http.Dir(dir))
}
