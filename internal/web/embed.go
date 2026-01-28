//go:build embed_web

package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var embeddedFiles embed.FS

// RegisterHandlers registers the static file handlers for the embedded web UI
func RegisterHandlers(r *gin.Engine) error {
	// Get the dist subdirectory from embedded files
	distFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return err
	}

	// Read index.html content for serving
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return err
	}

	// Serve static assets using StaticFS
	r.StaticFS("/assets", http.FS(mustSub(distFS, "assets")))

	// Serve the index.html for the root path
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	// Serve other static files (favicon, etc.)
	r.GET("/logo.svg", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "logo.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", data)
	})

	// SPA fallback - serve index.html for unmatched routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Don't serve index.html for API routes
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Don't serve index.html for swagger routes
		if strings.HasPrefix(path, "/swagger") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return nil
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
