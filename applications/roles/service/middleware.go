package service

import (
	"fmt"
	"net/http"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allowlisted origins
		allowedOrigins := map[string]bool{
			"http://localhost:3000":     true,
			"http://54.251.96.179:3000": true,
			"http://13.229.196.7:3000":  true,
		}

		// Log incoming origin
		fmt.Printf("CORS check - Origin: %s, Allowed: %v\n", origin, allowedOrigins[origin])

		// if allowedOrigins[origin] {
		// 	fmt.Println("Inside the if statment", origin)

		// 	w.Header().Set("Vary", "Origin")
		// }
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true") // ✅ REQUIRED for tokens

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
