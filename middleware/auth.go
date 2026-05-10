package middleware

import (
	"net/http"
	"strings"

	"project_smt6/auth"
	"project_smt6/domain"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID      = "user_id"
	ContextEmail       = "email"
	ContextRole        = "role"
	ContextWorkspaceID = "workspace_id"
)

func RequireAuth(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			tokenString = c.Query("access_token")
		}
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication token is required"})
			return
		}

		claims, err := authSvc.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication token is invalid"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)
		if claims.WorkspaceID != nil {
			c.Set(ContextWorkspaceID, *claims.WorkspaceID)
		}
		c.Next()
	}
}

func RequireRoles(roles ...domain.RoleName) gin.HandlerFunc {
	allowed := make(map[domain.RoleName]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		rawRole, ok := c.Get(ContextRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role is required"})
			return
		}
		role, ok := rawRole.(domain.RoleName)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role is invalid"})
			return
		}
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
			return
		}
		c.Next()
	}
}

func WorkspaceID(c *gin.Context) *uint {
	raw, ok := c.Get(ContextWorkspaceID)
	if !ok {
		return nil
	}
	workspaceID, ok := raw.(uint)
	if !ok {
		return nil
	}
	return &workspaceID
}

func UserID(c *gin.Context) *uint {
	raw, ok := c.Get(ContextUserID)
	if !ok {
		return nil
	}
	userID, ok := raw.(uint)
	if !ok {
		return nil
	}
	return &userID
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
