package webserver

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Webserver struct{}

func (w *Webserver) Start() error {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.StaticFile("/", "webserver/index.html")
	r.Static("/assets", "webserver/assets")
	r.GET("/ws", func(c *gin.Context) {
		socket(c.Writer, c.Request)
	})

	go r.Run()

	return nil
}

var upgrader = websocket.Upgrader{}

func socket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}
