package middleware

import "github.com/gin-gonic/gin"

func CompanyIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.GetHeader("X-Company-ID")
		if companyID != "" {
			c.Set("company_id", companyID)
		}
		c.Next()
	}
}
