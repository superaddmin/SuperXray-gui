package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
)

// DatabaseLeaseMiddleware drains active requests before a database import.
// The authenticated import handler obtains the exclusive lease itself.
func DatabaseLeaseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasSuffix(c.FullPath(), "/server/importDB") {
			release := database.AcquireReadLease()
			defer release()
		}
		c.Next()
	}
}
