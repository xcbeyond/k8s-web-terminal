package main

import (
	"k8s-web-terminal/ws"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.Use(static.Serve("/terminalui", static.LocalFile("./ui/terminal-ui", true)))
	router.Static("/terminalui", "./ui/terminal-ui")

	router.GET("/terminal", webTerminalHandle)

	router.Run(":8080")
}

func webTerminalHandle(c *gin.Context) {
	ws.WebSocketHandler(c.Writer, c.Request)
}
