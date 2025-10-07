//catches panics and returns 500 without crashing the server.

package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin" //gin context and middleware support 
)



//recovery protects the server from crashes if a panic occurs during request handling .
//it respond with 500 and logs the panic 


func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		//defer a function that recovers from panic if one happens during c.Next()
		defer func() {
			if r := recover(); r != nil { // if r is not nill , a panic occurred
				log.Printf("[panic] %v", r) //logthe panic valuee
				c.AbortWithStatusJSON(http.StatusInternalServerError, //return 500 json 
					gin.H{"error": "internal error"}) 
			}
		}()
		c.Next() // proceed to subsequent handlers ;; if one panics , defer above will handle it 
	}
}
