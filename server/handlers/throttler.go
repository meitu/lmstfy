package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/bitleak/lmstfy/throttler"
)

func Throttle(throttler *throttler.Throttler, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pool := c.GetString("pool")
		namespace := c.Param("namespace")
		queue := c.Param("queue")
		token := c.GetString("token")
		if action != "consume" && action != "produce" {
			c.Next()
			return
		}
		isRead := action == "consume"
		isReachRateLimited, err := throttler.IsReachRateLimit(pool, namespace, queue, token, isRead)
		if err != nil {
			logger := GetHTTPLogger(c)
			logger.WithFields(logrus.Fields{
				"token":  token,
				"action": action,
				"err":    err,
			}).Errorf("The throttler was broken")
			c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
			c.Abort()
			return
		}
		if isReachRateLimited {
			metrics.RateLimits.WithLabelValues(pool, namespace, token, action).Inc()
			msg := fmt.Sprintf("token(%s) %s reach the limit rate, please retry later", token, action)
			c.JSON(http.StatusTooManyRequests, gin.H{"msg": msg})
			c.Abort()
			return
		}
		c.Next()
		statusCode := c.Writer.Status()
		if (isRead && statusCode != http.StatusOK) || (!isRead && statusCode != http.StatusCreated) {
			throttler.RemedyLimiter(pool, namespace, queue, token, isRead)
		}
	}
}
