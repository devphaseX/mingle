package main

import (
	"net/http"
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
