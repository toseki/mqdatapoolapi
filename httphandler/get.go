package httphandler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"github.com/toseki/mqttdatapoolapi/storage"
)

type Handler interface {
	GetData(c echo.Context) error
}

type httpHandler struct {
	kvsh storage.Handler
}

func NewhttpHandler(kvsh storage.Handler) Handler {
	h := httpHandler{
		kvsh: kvsh,
	}
	return &h
}

// MainPage test response
func (h *httpHandler) GetData(c echo.Context) error {
	//username, _, _ := c.Request().BasicAuth() // get BasicAuth username info

	//test param.
	username := "demo" //lora/demo/000b78fffe051ccb/rx

	//topic := "lora/demo/000b78fffe051ccb/rx"
	topic := c.Param("base") + c.Param("userparam") + "/" + c.Param("param1") + "/" + c.Param("param2")

	payload, err := h.kvsh.GetPayloadData(topic)
	if err != nil {
		payload = []byte("no data")
	}
	log.Debug("kvs payload:", payload)

	if username != c.Param("userparam") {
		return c.String(http.StatusNotAcceptable, "not valid user.")
	}
	return c.String(http.StatusOK, fmt.Sprintf("%s", payload))
}
