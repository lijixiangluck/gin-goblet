package goblet

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lijixiangluck/gin-goblet/httpd"
	"net/http"
)

func Init() *Engine {
	g := gin.New()
	g.GET("/index")
	return g
}

func Run(handler http.Handler) {
	err := httpd.New().ListenAndServe("127.0.0.1:8080", handler)
	fmt.Println(err)
}
