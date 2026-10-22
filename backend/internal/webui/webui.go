// Package webui 托管前端静态文件（SPA fallback）。
package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"gost-panel/internal/resp"
)

// Register 把 staticDir 挂到 NoRoute：存在的文件直接返回，
// 其余 GET 路径 fallback 到 index.html（前端路由），非 GET 与 /api/ 返回统一 404 响应。
func Register(r *gin.Engine, staticDir string) {
	st, err := os.Stat(staticDir)
	if err != nil || !st.IsDir() {
		return
	}
	indexHTML := filepath.Join(staticDir, "index.html")
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet && !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			p := filepath.Join(staticDir, filepath.Clean(c.Request.URL.Path))
			if _, err := os.Stat(p); err == nil {
				c.File(p)
				return
			}
			c.File(indexHTML)
			return
		}
		resp.ErrNotFoundResp(c, "Not Found")
	})
}
