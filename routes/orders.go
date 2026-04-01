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

func addOrderRoutes(r *mux.Router, m *middleware.Middleware, serviceRegistry string) {
	ctx := context.Background()
	orderService, err := cache.Client().Get(ctx, "ORDERS-SERVICE").Result()
	if err != nil {
		eurekaConn := fargo.NewConn(serviceRegistry)
		orderService, err = getServiceURL(eurekaConn, "ORDERS-SERVICE")
		if err != nil {
			logger.Log().Infof("unable to get service url : %v", err)
		}

		cache.Client().Set(ctx, "ORDERS-SERVICE", orderService, 5*time.Minute)
	}

	orderRoutes := r.PathPrefix("/api/orders").Subrouter()
	orderRoutes.Use(m.RateLimitingMiddleware)
	orderRoutes.Use(m.AuthenticationMiddleware)
	orderRoutes.Use(m.CachingMiddleware)
	orderRoutes.HandleFunc("/{path:.*}", newProxy(orderService)).Methods(GET, POST, PUT, DELETE)
}
