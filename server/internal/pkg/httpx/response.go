package httpx

import "github.com/gin-gonic/gin"

type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func OK(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data})
}

func Fail(c *gin.Context, status int, err string) {
	c.AbortWithStatusJSON(status, Envelope{Success: false, Error: err})
}
