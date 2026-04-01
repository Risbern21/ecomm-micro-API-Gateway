package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hudl/fargo"
	"github.com/risbern21/api_gateway/internal/cache"
	"github.com/risbern21/api_gateway/internal/database"
	"github.com/risbern21/api_gateway/internal/logger"
	"github.com/risbern21/api_gateway/internal/migrations"
	"github.com/risbern21/api_gateway/routes"
)

func main() {
	logger.InitLogger()
	database.Setup()
	migrations.AutoMigrate()
	cache.Connect()

	server := createServer()

	if err := runServer(context.Background(), server, 3*time.Second); err != nil {
		log.Fatalf("unable to start the server : %v", err)
	}

}

func createServer() *http.Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":6969"
	}
	serviceRegistry := os.Getenv("SERVICE_REGISTRY")
	eurekaHostname := os.Getenv("EUREKA_HOSTNAME")
	if serviceRegistry == "" || eurekaHostname == "" {
		logger.Log().Fatal("env var not defined")
	}

	c := fargo.NewConn(serviceRegistry)
	instance := fargo.Instance{
		InstanceId:       "api-gateway",
		HostName:         eurekaHostname,
		App:              "API-GATEWAY",
		IPAddr:           "localhost",
		VipAddress:       "API-GATEWAY",
		SecureVipAddress: "API-GATEWAY",
		Status:           fargo.UP,
		Port:             6969,
		PortEnabled:      true,
		DataCenterInfo: fargo.DataCenterInfo{
			Name: fargo.MyOwn,
		},
		LeaseInfo: fargo.LeaseInfo{
			RenewalIntervalInSecs: 30,
			DurationInSecs:        90,
		},
	}

	// Register with Eureka
	err := c.RegisterInstance(&instance)
	if err != nil {
		log.Fatal("Failed to register:", err)
	}

	go heartBeat(c, instance)

	mux := mux.NewRouter()

	routes.AddRoutes(mux, serviceRegistry)

	return &http.Server{
		Addr:    port,
		Handler: mux,
	}
}

func runServer(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	serverErr := make(chan error, 1)

	go func() {
		log.Printf("API Gateway runnning on port %s", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-stop:
		log.Println("Shutdown Signal received")
	case <-ctx.Done():
		log.Println("Parent context cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if shutdownErr := server.Close(); shutdownErr != nil {
			return errors.Join(err, shutdownErr)
		}
		return err
	}

	log.Println("Server exited gracefully")
	return nil
}

func heartBeat(conn fargo.EurekaConnection, instance fargo.Instance) {
	for {
		err := conn.HeartBeatInstance(&instance)
		if err != nil {
			log.Println("Heartbeat failed:", err)
		} else {
			log.Println("Heartbeat sent")
		}

		time.Sleep(30 * time.Second)
	}
}
