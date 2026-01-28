package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Единая модель курса для всех операций
type Course struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	RepoTemplate string `json:"repoTemplate"`
	Description  string `json:"description"`
	Url          string `json:"url"` // Добавляем поле для списка
}

// База данных курсов (единый источник данных)
var courseDB = map[uint64]Course{
	1: {
		ID:           1,
		Name:         "hse-go",
		Status:       "in_progress",
		StartDate:    "2024-10-01",
		EndDate:      "2024-12-20",
		RepoTemplate: "git@gitlab.local/hse-go-template.git",
		Description:  "Курс по программированию на Go для студентов ВШЭ.",
		Url:          "https://fcstask.hse.ru/hse-go",
	},
	2: {
		ID:           2,
		Name:         "hse-rust",
		Status:       "finished",
		StartDate:    "2024-09-01",
		EndDate:      "2024-11-30",
		RepoTemplate: "git@gitlab.local/hse-rust-template.git",
		Description:  "Введение в язык Rust и системное программирование.",
		Url:          "https://fcstask.hse.ru/hse-rust",
	},
	3: {
		ID:           3,
		Name:         "hse-python",
		Status:       "in_progress",
		StartDate:    "2024-10-15",
		EndDate:      "2025-01-15",
		RepoTemplate: "git@gitlab.local/hse-python-template.git",
		Description:  "Основы Python и анализ данных.",
		Url:          "https://fcstask.hse.ru/hse-python",
	},
}

// GET `/api/courses` - Получить все курсы
func GetCoursesHandler(ctx echo.Context) error {
	courses := make([]Course, 0, len(courseDB))
	
	for _, course := range courseDB {
		courses = append(courses, course)
	}
	
	return ctx.JSON(http.StatusOK, courses)
}

// GET `/api/courses/:courseId` - Получить курс по ID
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


// POST `/api/courses`
type PostCourseModel struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	RepoTemplate string `json:"repoTemplate"`
	Description  string `json:"description"`
}

func PostCourseHandler(ctx echo.Context) error {
//добавить проверку на существование такого куса, если такой курс уже есть, то создавать такой же не нужно
  var req PostCourseModel
  if err := ctx.Bind(&req); err != nil {
    return ctx.JSON(http.StatusBadRequest, map[string]string{
      "error": "invalid JSON payload",
    })
  }
// нужно проверить все 7 полей
  if req.Name == "" || req.Slug == "" || req.Status == "" {
    return ctx.JSON(http.StatusBadRequest, map[string]string{
      "error": "name, slug, and status are required",
    })
  }

  var newID uint64 = 1
  for newID <= uint64(len(courseDB)) {
    if _, exists := courseDB[newID]; !exists {
      break
    }
    newID++
  }

  course := Course{
    ID:           newID,
    Name:         req.Name,
    Status:       req.Status,
    StartDate:    req.StartDate,
    EndDate:      req.EndDate,
    RepoTemplate: req.RepoTemplate,
    Description:  req.Description,
  }

  courseDB[newID] = course

  return ctx.JSON(http.StatusCreated, course)
}


// PUT `/api/courses/:courseId` - Обновить курс
func PutCourseHandler(ctx echo.Context) error {
	courseIDStr := ctx.Param("courseId")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 64)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid course ID",
		})
	}

	course, exists := courseDB[courseID]
	if !exists {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "course not found",
		})
	}

	var req PostCourseModel
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to parse request body as JSON",
		})
	}

	if req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
	}

	// Обновляем курс, сохраняя поля которые не переданы в запросе
	updatedCourse := Course{
		ID:           courseID,
		Name:         req.Name,
		Status:       req.Status,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		RepoTemplate: req.RepoTemplate,
		Description:  req.Description,
	}
	
	// Если какие-то поля не переданы, оставляем старые значения
	if req.Status == "" {
		updatedCourse.Status = course.Status
	}
	if req.StartDate == "" {
		updatedCourse.StartDate = course.StartDate
	}
	if req.EndDate == "" {
		updatedCourse.EndDate = course.EndDate
	}
	if req.RepoTemplate == "" {
		updatedCourse.RepoTemplate = course.RepoTemplate
	}
	if req.Description == "" {
		updatedCourse.Description = course.Description
	}

	courseDB[courseID] = updatedCourse
	return ctx.JSON(http.StatusOK, updatedCourse)
}