package routes

import (
	"context"
	"time"

	"github.com/gorilla/mux"
	"github.com/hudl/fargo"
	"github.com/risbern21/api_gateway/internal/cache"
	"github.com/risbern21/api_gateway/internal/logger"
	"github.com/risbern21/api_gateway/internal/middleware"
)

func addChatRoutes(r *mux.Router, m *middleware.Middleware, serviceRegistry string) {
	ctx := context.Background()
	chatService, err := cache.Client().Get(ctx, "CHAT-CLIENT-SERVICE").Result()
	if err != nil {
		eurekaConn := fargo.NewConn(serviceRegistry)
		chatService, err = getServiceURL(eurekaConn, "CHAT-CLIENT-SERVICE")
		if err != nil {
			logger.Log().Infof("unable to get service url : %v", err)
		}

		cache.Client().Set(ctx, "CHAT-CLIENT-SERVICE", chatService, 5*time.Minute)
	}

	chatClientRoutes := r.PathPrefix("/api/chat").Subrouter()
	chatClientRoutes.Use(m.RateLimitingMiddleware)
	chatClientRoutes.Use(m.AuthenticationMiddleware)
	chatClientRoutes.Use(m.CachingMiddleware)
	chatClientRoutes.HandleFunc("/{path:.*}", newProxy(chatService)).Methods(GET, POST, PUT, DELETE)
}

func addGenerationRoutes(r *mux.Router, m *middleware.Middleware, serviceRegistry string) {
	ctx := context.Background()
	generateService, err := cache.Client().Get(ctx, "CHAT-CLIENT-SERVICE").Result()
	if err != nil {
		serviceRegistry := serviceRegistry

		eurekaConn := fargo.NewConn(serviceRegistry)
		generateService, err = getServiceURL(eurekaConn, "CHAT-CLIENT-SERVICE")
		if err != nil {
			logger.Log().Infof("unable to get service url : %v", err)
		}

		cache.Client().Set(ctx, "CHAT-CLIENT-SERVICE", generateService, 5*time.Minute)
	}

	chatClientRoutes := r.PathPrefix("/api/generate").Subrouter()
	chatClientRoutes.Use(m.RateLimitingMiddleware)
	chatClientRoutes.Use(m.AuthenticationMiddleware)
	chatClientRoutes.Use(m.CachingMiddleware)
	chatClientRoutes.HandleFunc("/{path:.*}", newProxy(generateService)).Methods(GET, POST, PUT, DELETE)
}
