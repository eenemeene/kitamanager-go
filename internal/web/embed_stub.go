//go:build !embed_web

package web

import (
	"github.com/gin-gonic/gin"
)

// RegisterHandlers is a stub when web assets are not embedded.
// Use -tags=embed_web to build with embedded web assets.
func RegisterHandlers(r *gin.Engine) error {
	// No-op: web assets not embedded in this build
	return nil
}
