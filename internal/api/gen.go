//go:generate oapi-codegen -generate types,server -package api -o api.gen.go ../../api/openapi.yaml
//go:generate mockgen -source=api.gen.go -destination=server.gen.go -package=api

package api

import (
	"fcstask-backend/internal/server"

	handler "fcstask-backend/internal/server/handler"

	"github.com/labstack/echo/v4"
)

func RegisterHandlers(e *echo.Echo, apiServer *server.Server) {
	e.GET("/api/courses", handler.GetCoursesHandler)
}
