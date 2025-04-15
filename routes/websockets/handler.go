package websockets

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"github.com/tenkorangjr/circle-app/db"
	"github.com/tenkorangjr/circle-app/models"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var mRouter = MessageRouter{}
var validate = validator.New(validator.WithRequiredStructEnabled())

type Message struct {
	To  string `validate:"required,email" json:"to"`
	Msg string `validate:"required" json:"msg"`
}

type MessageRouter struct{}

func (m *MessageRouter) RouteMessage(msg Message) error {
	var receiver models.User
	if err := db.DB.Where("email = ?", msg.To).First(&receiver).Error; err != nil {
		return errors.New("no such email in database")
	}

	receiverClient, ok := WSManager.GetClient(receiver.ID)
	if !ok {
		zap.S().Infof("Receiver (%d) currently inactive", receiver.ID)
		return errors.New("receiver is currently inactive")
	}

	receiverClient.SendChan <- []byte(msg.Msg)
	return nil
}

func HandleWs(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		zap.S().Error(err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upgrade request"})
		return
	}
	defer conn.Close()

	userId := ctx.GetUint("userId")
	WSManager.AddClient(userId, conn)
	zap.S().Infof("User %d connected", userId)

	client, _ := WSManager.GetClient(userId)

	defer func() {
		WSManager.RemoveClient(userId)
		client.Conn.Close()
		zap.S().Infof("User %d disconnected", userId)
	}()

	go readPump(client)
	writePump(client)
}

func readPump(client *Client) {
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zap.S().Errorf("unexpected close error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			zap.S().Errorf("failed to unmarshal message: %v", err)
			continue
		}

		if err := validate.Struct(msg); err != nil {
			zap.S().Error("To address is not a valid email")
			continue
		}

		zap.S().Infof("Routing message to %s: %s", msg.To, msg.Msg)
		if err := mRouter.RouteMessage(msg); err != nil {
			zap.S().Errorf("failed to route message: %v", err)
		}
	}
}

func writePump(client *Client) {
	for msg := range client.SendChan {
		err := client.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			zap.S().Errorf("failed to write message: %v", err)
			break
		}
	}
}
