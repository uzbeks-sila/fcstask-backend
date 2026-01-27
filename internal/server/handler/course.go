package handler

import (
	"net/http"
	"strconv"

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

// GET `/api/courses/:courseId`
type GetResponseCourseIdModel struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	RepoTemplate string `json:"repoTemplate"`
	Description  string `json:"description"`
}

var courseDB = map[uint64]GetResponseCourseIdModel{
	1: {
		ID:           1,
		Name:         "hse-go",
		Status:       "in_progress",
		StartDate:    "2024-10-01",
		EndDate:      "2024-12-20",
		RepoTemplate: "git@gitlab.local/hse-go-template.git",
		Description:  "Курс по программированию на Go для студентов ВШЭ.",
	},
	2: {
		ID:           2,
		Name:         "hse-rust",
		Status:       "finished",
		StartDate:    "2024-09-01",
		EndDate:      "2024-11-30",
		RepoTemplate: "git@gitlab.local/hse-rust-template.git",
		Description:  "Введение в язык Rust и системное программирование.",
	},
	3: {
		ID:           3,
		Name:         "hse-python",
		Status:       "in_progress",
		StartDate:    "2024-10-15",
		EndDate:      "2025-01-15",
		RepoTemplate: "git@gitlab.local/hse-python-template.git",
		Description:  "Основы Python и анализ данных.",
	},
}

func GetCourseIdHandler(ctx echo.Context) error {
	courseIDStr := ctx.Param("courseId")

	courseID, err := strconv.ParseUint(courseIDStr, 10, 64)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid course ID, must be a number",
		})
	}

	course, exists := courseDB[courseID]
	if !exists {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "course not found",
		})
	}

	return ctx.JSON(http.StatusOK, course)
}