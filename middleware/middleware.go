package middleware

import (
	"log"
	"net/http"
	"time"

	"flugo.com/router"
)

func CORS() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}
}

func Logger() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next(w, r)
			duration := time.Since(start)
			log.Printf("[%s] %s %s - %v", r.Method, r.URL.Path, r.RemoteAddr, duration)
		}
	}
}

func JSONContentType() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next(w, r)
		}
	}
}

func Recovery() router.MiddlewareFunc {
	return func(next router.HandlerFunc) router.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next(w, r)
		}
	}
}
