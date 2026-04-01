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

func addProductRoutes(r *mux.Router, m *middleware.Middleware, serviceRegistry string) {
	ctx := context.Background()
	productsService, err := cache.Client().Get(ctx, "PRODUCTS-SERVICE").Result()
	if err != nil {
		eurekaConn := fargo.NewConn(serviceRegistry)
		productsService, err = getServiceURL(eurekaConn, "PRODUCTS-SERVICE")
		if err != nil {
			logger.Log().Infof("unable to get service url : %v", err)
		}

		cache.Client().Set(ctx, "PRODUCTS-SERVICE", productsService, 5*time.Minute)
	}

	productRoutes := r.PathPrefix("/api/products").Subrouter()
	productRoutes.Use(m.RateLimitingMiddleware)
	productRoutes.Use(m.AuthenticationMiddleware)
	productRoutes.Use(m.CachingMiddleware)
	productRoutes.HandleFunc("/{path:.*}", newProxy(productsService)).Methods(GET, POST, PUT, DELETE)
}
