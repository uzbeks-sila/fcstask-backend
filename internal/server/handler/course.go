package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GET `/api/courses`
type GetResponseCourseModel struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Url    string `json:"url"`
}

func GetCoursesHandler(ctx echo.Context) error {
	courses := []GetResponseCourseModel{
		{
			ID:     1,
			Name:   "hse-go",
			Status: "in_progress",
			Url:    "https://fcstask.hse.ru/hse-go",
		},
		{
			ID:     2,
			Name:   "hse-rust",
			Status: "finished",
			Url:    "https://fcstask.hse.ru/hse-rust",
		},
		{
			ID:     3,
			Name:   "hse-python",
			Status: "in_progress",
			Url:    "https://fcstask.hse.ru/hse-python",
		},
	}
	return ctx.JSON(http.StatusOK, courses)
}

// POST `/api/courses`
