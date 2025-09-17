package server

import "github.com/gin-gonic/gin"

type Handler interface {
	SetupRoute(router gin.IRouter)

	Handle(c *gin.Context)
}
