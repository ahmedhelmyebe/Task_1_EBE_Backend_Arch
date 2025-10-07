// simple structured request logging

package middlewares

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

//RequestLogger prints method , path , status and duration for each request 
//simple and effective for local dev ;;; in prod you might use structured logging 
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now() //erecord start time
		path := c.Request.URL.Path //// Keep the path for logging (useful after c.Next()).
		c.Next() // Run downstream handlers/middlewares.
		log.Printf("%s %s %d %s",  //log  linee
		c.Request.Method, //http method (get , POST ,etc ...)
		 path, //request path 
		  c.Writer.Status(), //final status code
		  time.Since(start)) //elapsed time
	}
}
