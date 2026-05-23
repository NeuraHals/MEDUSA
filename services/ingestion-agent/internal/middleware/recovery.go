package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Recovery catches panics and returns a 500 without crashing the server.
func Recovery(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Path()),
				)
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
