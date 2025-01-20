package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
)

func (app *application) AuthMiddleware() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// tokenString := c.GetHeader("Authorization")
		// if tokenString == "" {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		// 	c.Abort()
		// 	return
		// }

		// claims, err := tokenService.ValidateAccessToken(tokenString)
		// if err != nil {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		// 	c.Abort()
		// 	return
		// }

		// c.Set("user_id", claims.UserID)
		// c.Set("session_id", claims.SessionID)
		// c.Next()
	})
}

func (app *application) BasicAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				app.authenticationBasicRequiredResponse(w, r, "authorization header is missing or empty")
				return
			}

			parts := strings.Split(authHeader, " ")

			if !(len(parts) == 2 && parts[0] == "Basic") {
				app.authenticationBasicRequiredResponse(w, r, "authorization header is malformed")
				return
			}

			decodedToken, err := base64.StdEncoding.DecodeString(parts[1])

			if err != nil {
				app.authenticationBasicRequiredResponse(w, r, err.Error())
				return
			}

			creds := bytes.SplitN(decodedToken, []byte(":"), 2)

			if !(len(creds) == 2 &&
				app.config.auth.basic.username == string(creds[0]) &&
				app.config.auth.basic.password == string(creds[1])) {
				app.authenticationBasicRequiredResponse(w, r, "invalid credentials username or password")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
