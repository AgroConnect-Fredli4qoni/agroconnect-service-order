package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/agroconnect/service-order/handlers"
	"github.com/agroconnect/service-order/repository"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "agro_user:agro_password@tcp(localhost:3306)/agroconnect_db?parseTime=true"
	}

	catalogURL := os.Getenv("CATALOG_SERVICE_URL")
	if catalogURL == "" {
		catalogURL = "http://localhost:8081"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "agroconnect_super_secure_jwt_secret_key_2026"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to initialize MySQL driver: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("Warning: MySQL ping check failed: %v", err)
	} else {
		log.Println("Successfully connected to MySQL database")
	}

	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	authHandler := handlers.NewAuthHandler(userRepo, jwtSecret)
	orderHandler := handlers.NewOrderHandler(orderRepo, catalogURL)

	router := mux.NewRouter()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service":  "agroconnect-service-order",
			"status":   "UP",
			"database": "MySQL (ACID)",
			"version":  "1.0.0",
		})
	}).Methods("GET")

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")
	api.HandleFunc("/orders/user", orderHandler.GetOrdersByUser).Methods("GET")
	api.HandleFunc("/orders/{code}", orderHandler.GetOrderByCode).Methods("GET")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler(router)

	log.Printf("AgroConnect Service Order running on port :%s", port)
	if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
		log.Fatalf("Server terminated: %v", err)
	}
}
