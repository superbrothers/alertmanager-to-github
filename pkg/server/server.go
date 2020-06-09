package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/pfnet-research/alertmanager-to-github/pkg/notifier"
	"github.com/pfnet-research/alertmanager-to-github/pkg/types"
	"net/http"
)

type Server struct {
	Notifier notifier.Notifier
}

func New(notifier notifier.Notifier) (*Server) {
	return &Server{
		Notifier: notifier,
	}
}

func (s *Server) Router() *gin.Engine {
	router := gin.Default()
	router.POST("/v1/webhook", s.v1Webhook)

	return router
}

func (s *Server) v1Webhook(c *gin.Context) {
	payload := &types.WebhookPayload{}

	if err := c.ShouldBindJSON(payload); err != nil {
		log.Error().Err(err).Msg("error binding JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Debug().Interface("payload", payload).Msg("/v1/webhook")

	ctx := context.TODO()
	if err := s.Notifier.Notify(ctx, payload); err != nil {
		log.Error().Err(err).Msg("error notifying")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}