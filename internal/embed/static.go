package embed

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register 挂载 SPA 三入口和静态资源。
func Register(r *gin.Engine) {
	root, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	r.GET("/", serveFile(root, "index.html"))
	r.GET("/embed", serveFile(root, "embed.html"))
	r.GET("/admin", serveFile(root, "admin.html"))

	assetsFS, _ := fs.Sub(root, "assets")
	assetsHandler := http.FileServer(http.FS(assetsFS))
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.StripPrefix("/assets", assetsHandler).ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		if data, err := fs.ReadFile(root, "favicon.ico"); err == nil {
			c.Data(200, "image/x-icon", data)
			return
		}
		c.Status(204)
	})
}

func serveFile(root fs.FS, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := fs.ReadFile(root, name)
		if err != nil {
			c.String(404, "not found: %s", name)
			return
		}
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Data(200, "text/html; charset=utf-8", data)
	}
}
