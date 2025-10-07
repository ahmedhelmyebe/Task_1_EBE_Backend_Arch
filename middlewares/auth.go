// validates JWT and injects ->
// uid into Gin context for downstream handlers.

package middlewares

import (
	"net/http"
	"strconv" // Convert string claim to int when needed.

	"HelmyTask/global" // For the context key to store user ID.

	"github.com/gin-gonic/gin"     // Gin context/request/response types
	"github.com/golang-jwt/jwt/v5" // JWT parsing and validation
)

// Auth returns a Gin middleware that validates "Authorization: Bearer <token>"
// and injects the user ID ("uid") into the request context if the token is valid.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) { // Middleware function closure captures jwtSecret. 
		auth := c.GetHeader("Authorization") //read authorization header from request
		// Quick check : must start with "bearer" and be long 
		if len(auth) < 8 || auth[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return //stop processing further handlers 
		}
		raw := auth[7:] //extract the token substring after "Bearer"

		// parse and validate token signature using the shared secret
		t, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil 
		})
		//reject with 401 if the token is not valid or if an error exist 
		if err != nil || !t.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		//we expect MapClaims (string any map) to exract tored fields 
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		// extract subject (user ID) from the claims and normalize its type 
		sub := claims["sub"]
		switch v := sub.(type) {
		case float64: // JSON numbers often decode to float64; cast to uint.
			c.Set(global.CtxUserIDKey, uint(v))
		case string: // Sometimes IDs may be strings; try to parse.
			if n, err := strconv.Atoi(v); err == nil {
				c.Set(global.CtxUserIDKey, uint(n))
			}
		}
		c.Next() // Continue to the actual handler. 
	}
}
