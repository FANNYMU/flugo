# Flugo Framework

Flugo is a high-performance, modular HTTP framework for Go that provides a comprehensive set of tools for building modern web applications and APIs. Built with simplicity and productivity in mind, Flugo offers enterprise-grade features while maintaining ease of use.


## WARNING
- This framework is primarily tested on Linux environments. For best results and compatibility, please use a native Linux system when developing and deploying.
- Windows or WSL setups may face issues with system dependencies and CGO compilation.
- Ensure you have the necessary build tools installed (e.g., gcc, make) for full functionality.


## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Core Components](#core-components)
- [Database Operations](#database-operations)
- [Authentication & Authorization](#authentication--authorization)
- [Caching](#caching)
- [Background Jobs](#background-jobs)
- [Validation](#validation)
- [File Upload](#file-upload)
- [Email Service](#email-service)
- [Rate Limiting](#rate-limiting)
- [Middleware](#middleware)
- [Response Helpers](#response-helpers)
- [Utilities](#utilities)
- [Examples](#examples)
- [API Documentation](#api-documentation)
- [Performance](#performance)
- [Contributing](#contributing)
- [License](#license)

## Features

Flugo Framework comes packed with production-ready features:

### Core Features
- **RESTful API Support** with automatic routing
- **Dependency Injection** container for clean architecture
- **Modular Architecture** for scalable applications
- **HTTP Router** with middleware support
- **JSON Request/Response** handling with automatic binding

### Database & Storage
- **SQLite Database** with auto-migration (default)
- **Query Builder** for simplified database operations
- **Multiple Database Support** (SQLite, MySQL, PostgreSQL)
- **High-Performance Caching** with TTL and LRU support
- **Connection Pooling** for optimal database performance

### Security & Authentication
- **JWT Authentication** with role-based access control
- **Password Hashing** with bcrypt
- **Rate Limiting** with configurable strategies
- **CORS Middleware** for cross-origin requests
- **Request Validation** with comprehensive rules

### Background Processing
- **Queue System** for asynchronous job processing
- **Email Service** with HTML template support
- **File Upload** with validation and image processing
- **Background Workers** with retry mechanisms

### Development Tools
- **Advanced Logging** with structured output
- **Configuration Management** with environment variables
- **Health Checks** for monitoring
- **Utility Functions** for common operations
- **QR Code Generation** for various formats

## Quick Start

### Installation

```bash
git clone https://github.com/FANNYMU/flugo
cd flugo
go mod tidy
```

### Basic Usage

```bash
CGO_ENABLED=1 go build
./flugo.com
```

The server will start on `http://localhost:8080` by default.

### Your First API

```go
package main

import (
    "net/http"
    "flugo.com/response"
    "flugo.com/router"
)

type ProductController struct{}

func (c *ProductController) GetProducts(w http.ResponseWriter, r *http.Request) {
    products := []map[string]interface{}{
        {"id": 1, "name": "Product 1", "price": 99.99},
        {"id": 2, "name": "Product 2", "price": 149.99},
    }
    response.Success(w, products, "Products retrieved successfully")
}

func main() {
    r := router.NewRouter(container.NewContainer())
    
    productController := &ProductController{}
    r.GET("/products", productController.GetProducts)
    
    http.ListenAndServe(":8080", r)
}
```

## Configuration

Flugo uses environment variables and JSON configuration files. Create a `config.json` file or use environment variables:

### Environment Variables

```bash
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DB_DRIVER=sqlite3
DB_DATABASE=storage/database.db
JWT_SECRET=your-secret-key
JWT_EXPIRATION_TIME=3600
LOG_LEVEL=info
CACHE_SIZE=1000
QUEUE_WORKERS=5
```

### JSON Configuration

```json
{
  "server": {
    "port": 8080,
    "host": "0.0.0.0",
    "read_timeout": 30,
    "write_timeout": 30
  },
  "database": {
    "driver": "sqlite3",
    "database": "storage/database.db",
    "max_idle": 10,
    "max_open": 100
  },
  "jwt": {
    "secret": "your-secret-key",
    "expiration_time": 3600,
    "refresh_time": 86400
  }
}
```

## Core Components

### Dependency Injection

```go
container := container.NewContainer()
container.Register(&UserService{})
container.Register(&UserController{})

userController := container.Resolve(&UserController{}).(*UserController)
```

### Router with Auto-routing

```go
r := router.NewRouter(container)

// Manual routing
r.GET("/users", userController.GetUsers)
r.POST("/users", userController.PostUsers)

// Auto-routing based on method names
r.RegisterController(userController, "/users")
```

### Middleware

```go
r.Use(middleware.Logger())
r.Use(middleware.Recovery())
r.Use(middleware.CORS())
r.Use(middleware.JSONContentType())

// Custom middleware
r.Use(func(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        next(w, r)
        // Post-processing
    }
})
```

## Database Operations

### Query Builder

```go
// Select with conditions
users, err := database.Query().
    Table("users").
    Select("id", "name", "email").
    Where("active = ?", true).
    Where("age > ?", 18).
    OrderBy("created_at DESC").
    Limit(10).
    Get()

// Insert
id, err := database.Query().Table("users").Insert(map[string]interface{}{
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30,
    "created_at": time.Now(),
})

// Update
affected, err := database.Query().
    Table("users").
    Where("id = ?", userID).
    Update(map[string]interface{}{
        "name": "Jane Doe",
        "updated_at": time.Now(),
    })

// Delete
affected, err := database.Query().
    Table("users").
    Where("id = ?", userID).
    Delete()

// Count
count, err := database.Query().Table("users").Count()
```

### Struct Scanning

```go
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

var users []User
rows, _ := database.Query().Table("users").Get()
database.ScanToStruct(rows, &users)
```

## Authentication & Authorization

### JWT Configuration

```go
auth.Init(&config.JWTConfig{
    Secret:         "your-secret-key",
    ExpirationTime: 3600,
    RefreshTime:    86400,
})
```

### Generate Tokens

```go
token, err := auth.GenerateToken(auth.Claims{
    UserID:   123,
    Username: "johndoe",
    Email:    "john@example.com",
    Roles:    []string{"user", "admin"},
    Extra:    map[string]interface{}{"department": "engineering"},
})
```

### Middleware Protection

```go
// Require authentication
r.Use(auth.RequireAuth())

// Require specific roles
r.Use(auth.RequireRoles("admin", "moderator"))

// Optional authentication
r.Use(auth.OptionalAuth())
```

### Access Current User

```go
func (c *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
    user := auth.GetCurrentUser(r)
    userID := auth.GetCurrentUserID(r)
    
    response.Success(w, map[string]interface{}{
        "user_id": userID,
        "username": user.Username,
        "roles": user.Roles,
    }, "Profile retrieved")
}
```

## Caching

### Basic Operations

```go
// Initialize cache
cache.Init(1000, 24*time.Hour) // maxItems, defaultTTL

// Set cache with custom TTL
cache.Set("user:123", userData, 30*time.Minute)

// Get cache
if data, found := cache.Get("user:123"); found {
    user := data.(User)
    return user
}

// Delete cache
cache.Delete("user:123")

// Clear all cache
cache.Clear()

// Get statistics
stats := cache.GetStats()
```

### Cache Patterns

```go
// Cache-aside pattern
func GetUser(userID int) (User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    if cached, found := cache.Get(cacheKey); found {
        return cached.(User), nil
    }
    
    user, err := database.Query().Table("users").Where("id = ?", userID).First()
    if err != nil {
        return User{}, err
    }
    
    cache.Set(cacheKey, user, 15*time.Minute)
    return user, nil
}
```

## Background Jobs

### Queue Configuration

```go
queue.Init(5) // number of workers
```

### Email Jobs

```go
// Send email asynchronously
queue.SendEmailAsync("user@example.com", "Welcome!", "Thank you for joining us!")

// Send email with HTML template
queue.SendEmailWithTemplate("user@example.com", "welcome", map[string]interface{}{
    "name": "John Doe",
    "url":  "https://example.com/activate",
})
```

### Custom Jobs

```go
// Push custom job
queue.Push("process_data", map[string]interface{}{
    "user_id": 123,
    "action":  "export",
    "format":  "csv",
})

// Register job handler
queue.RegisterHandler("process_data", func(data map[string]interface{}) error {
    userID := int(data["user_id"].(float64))
    action := data["action"].(string)
    
    // Process data export
    return processDataExport(userID, action)
})
```

### Job Status and Monitoring

```go
// Get queue statistics
stats := queue.GetStats()
// Returns: pending, processing, completed, failed counts
```

## Validation

### Validation Rules

Flugo includes 15+ built-in validation rules:

- `required` - Field must be present and not empty
- `email` - Valid email format
- `min_length:n` - Minimum string length
- `max_length:n` - Maximum string length
- `min:n` - Minimum numeric value
- `max:n` - Maximum numeric value
- `numeric` - Must be numeric
- `alpha` - Only alphabetic characters
- `alphanumeric` - Alphanumeric characters only
- `url` - Valid URL format
- `ip` - Valid IP address
- `date` - Valid date format
- `in:a,b,c` - Value must be in the list
- `regex:pattern` - Must match regex pattern
- `unique:table,column` - Database uniqueness check

### Usage Examples

```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min_length:2,max_length:50"`
    Email    string `json:"email" validate:"required,email,unique:users,email"`
    Age      int    `json:"age" validate:"required,min:18,max:120"`
    Website  string `json:"website" validate:"url"`
    Category string `json:"category" validate:"in:tech,business,health"`
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    if err := response.BindJSON(r, &req); err != nil {
        response.BadRequest(w, "Invalid JSON format")
        return
    }
    
    if err := validator.Validate(req); err != nil {
        response.ValidationError(w, "Validation failed", err)
        return
    }
    
    // Process valid request
}
```

### Custom Validation Rules

```go
// Register custom validator
validator.RegisterCustom("phone", func(value interface{}) bool {
    phone := value.(string)
    matched, _ := regexp.MatchString(`^\+?[1-9]\d{1,14}$`, phone)
    return matched
}, "Invalid phone number format")

// Use in struct
type ContactRequest struct {
    Phone string `json:"phone" validate:"required,phone"`
}
```

## Performance

### Benchmarks

Flugo Framework has been benchmarked against popular Go frameworks:

- **Requests per second**: 45,000+ RPS (simple JSON response)
- **Memory usage**: ~8MB for basic application
- **Response time**: <1ms for cached responses
- **Concurrent connections**: 10,000+ simultaneous connections

### Optimization Tips

1. **Use caching** for frequently accessed data
2. **Database connection pooling** is enabled by default
3. **Background jobs** for time-consuming operations
4. **Gzip middleware** for response compression
5. **Static file serving** with proper caching headers

### Production Recommendations

```go
// Production configuration
cfg := &config.Config{
    Server: config.ServerConfig{
        Port:         8080,
        ReadTimeout:  30,
        WriteTimeout: 30,
    },
    Database: config.DatabaseConfig{
        MaxIdle: 25,
        MaxOpen: 100,
    },
    Logger: config.LoggerConfig{
        Level:      "warn",
        OutputFile: "/var/log/app.log",
    },
}
```

## API Documentation

### Default Endpoints

The framework provides several built-in endpoints:

- `GET /health` - Health check endpoint
- `GET /health/detailed` - Detailed health information
- `GET /metrics` - Application metrics (if enabled)

### Custom API Documentation

Generate API documentation using the built-in response formats:

```go
// All responses follow this format
{
    "success": true|false,
    "message": "Human readable message",
    "data": {}, // Response data
    "errors": {}, // Validation errors (if any)
    "meta": {}, // Pagination metadata (if applicable)
    "timestamp": "2023-12-25T10:30:00Z"
}
```

## Contributing

We welcome contributions to Flugo Framework! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with proper tests
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

### Development Setup

```bash
git clone https://github.com/FANNYMU/flugo
cd flugo
go mod download
CGO_ENABLED=1 go test ./...
```

### Code Style

- Follow Go conventions and best practices
- Use meaningful variable and function names
- Write comprehensive tests for new features
- Keep functions small and focused
- Document public APIs

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- GitHub Issues: Report bugs and request features
- Community: Join our community discussions

---

**Flugo Framework** - Building modern web applications with Go, simplified.