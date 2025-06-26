package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flugo.com/auth"
	"flugo.com/cache"
	"flugo.com/config"
	"flugo.com/container"
	"flugo.com/database"
	"flugo.com/logger"
	"flugo.com/middleware"
	"flugo.com/queue"
	"flugo.com/ratelimit"
	"flugo.com/response"
	"flugo.com/router"
	"flugo.com/validator"
)

// Example User model
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Example DTO for validation
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min_length:2,max_length:100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min_length:6"`
}

// Example controller - This is where you can code your APIs!
type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

// GET /users - Auto-routing magic!
func (c *UserController) GetUsers(w http.ResponseWriter, r *http.Request) {
	// This is where you write your logic!
	// Example: Get users from database
	rows, err := database.Query().Table("users").Get()
	if err != nil {
		response.InternalError(w, "Failed to fetch users")
		return
	}
	defer rows.Close()

	var users []User
	err = database.ScanToStruct(rows, &users)
	if err != nil {
		response.InternalError(w, "Failed to scan users")
		return
	}

	response.Success(w, users, "Users retrieved successfully")
}

// POST /users - Create new user
func (c *UserController) PostUsers(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	// Parse and validate request
	if err := response.BindJSON(r, &req); err != nil {
		response.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate DTO
	if err := validator.Validate(req); err != nil {
		response.ValidationError(w, "Validation failed", err)
		return
	}

	// Create user in database
	id, err := database.Query().Table("users").Insert(map[string]interface{}{
		"name":       req.Name,
		"email":      req.Email,
		"password":   req.Password, // Should be hashed in production
		"created_at": time.Now(),
	})
	if err != nil {
		response.InternalError(w, "Failed to create user")
		return
	}

	// Send welcome email asynchronously (if you want)
	queue.SendEmailAsync(req.Email, "Welcome!", "Thank you for joining us!")

	response.Created(w, map[string]interface{}{
		"id":    id,
		"name":  req.Name,
		"email": req.Email,
	}, "User created successfully")
}

func main() {
	log.Println("Starting Flugo Framework...")

	// Load configuration
	cfg := config.Load()

	// Create storage directory
	os.MkdirAll("storage", 0755)

	// Initialize core services
	logger.Init(&cfg.Logger)
	database.Init(&cfg.Database)
	cache.Init(1000, 24*time.Hour)
	validator.InitValidators()

	// Initialize JWT
	auth.Init(&cfg.JWT)

	// Initialize queue
	if cfg.Queue.Enabled {
		queue.Init(cfg.Queue.Workers)
	}

	// Initialize rate limiter
	ratelimit.Init(100, time.Minute)

	// Setup DI container and router
	container := container.NewContainer()
	r := router.NewRouter(container)

	// Global middlewares
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.JSONContentType())

	// Register your controllers here with auto-routing!
	userController := NewUserController()
	r.RegisterController(userController, "/users")

	// Manual route untuk testing
	r.GET("/users", userController.GetUsers)
	r.POST("/users", userController.PostUsers)

	// Health check endpoint
	r.GET("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		}, "Service is healthy")
	})

	// Utility endpoints for fun 🎯
	r.GET("/utils/time", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]interface{}{
			"current_time": time.Now(),
			"unix":         time.Now().Unix(),
			"formatted":    time.Now().Format("2006-01-02 15:04:05"),
		}, "Current time")
	})

	r.POST("/utils/echo", func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		if err := response.BindJSON(r, &data); err != nil {
			response.BadRequest(w, "Invalid JSON")
			return
		}
		response.Success(w, data, "Echo response")
	})

	// Graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		log.Println("Shutting down gracefully...")
		if database.DefaultDB != nil {
			database.DefaultDB.Close()
		}
		log.Println("Flugo Framework stopped")
		os.Exit(0)
	}()

	// Print startup message
	log.Println("")
	log.Println("Flugo Framework is ready!")
	log.Printf("Server running on http://localhost:%d", cfg.Server.Port)
	log.Println("")
	log.Println("Available Endpoints:")
	log.Println("   GET    /health          - Health check")
	log.Println("   GET    /users           - Get all users")
	log.Println("   POST   /users           - Create user")
	log.Println("   GET    /utils/time      - Get current time")
	log.Println("   POST   /utils/echo      - Echo request")
	log.Println("")
	log.Println("This is your playground! Start coding in main.go")
	log.Println("Add your controllers, modify routes, have fun!")
	log.Println("")

	// Start server
	address := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := http.ListenAndServe(address, r); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
