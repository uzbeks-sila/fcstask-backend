package handler

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	MinTokenLength = 10
	MaxTokenLength = 500
	MinSlugLength  = 3
	MaxSlugLength  = 50
	MinNameLength  = 3
	MaxNameLength  = 100
	MaxDescLength  = 500
)

// CourseStatus - статус курса (соответствует фронтенду)
type CourseStatus string

const (
	CourseStatusCreated        CourseStatus = "created"
	CourseStatusHidden         CourseStatus = "hidden"
	CourseStatusInProgress     CourseStatus = "in_progress"
	CourseStatusAllTasksIssued CourseStatus = "all_tasks_issued"
	CourseStatusDoreshka       CourseStatus = "doreshka"
	CourseStatusFinished       CourseStatus = "finished"
)

// Course - основная модель курса (соответствует контракту)
type Course struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Status       CourseStatus `json:"status"`
	StartDate    string       `json:"startDate"`
	EndDate      string       `json:"endDate"`
	RepoTemplate string       `json:"repoTemplate"`
	Description  string       `json:"description"`
	URL          string       `json:"url"`
}

// PostCourseRequest - модель для создания курса
type PostCourseRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	RepoTemplate string `json:"repoTemplate"`
	Description  string `json:"description"`
}

// ValidationError - структура для ошибок валидации
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ВРЕМЕННАЯ БАЗА ДАННЫХ (in-memory storage)
var (
	courseDB = map[string]Course{
		"algorithms": {
			ID:           "algorithms",
			Name:         "Algorithms 101",
			Status:       CourseStatusInProgress,
			StartDate:    "2024-10-01",
			EndDate:      "2024-12-20",
			RepoTemplate: "git@gitlab.local/algorithms-template.git",
			Description:  "Основы алгоритмов и структур данных",
			URL:          "/course/algorithms",
		},
		"mlops": {
			ID:           "mlops",
			Name:         "MLOps Studio",
			Status:       CourseStatusAllTasksIssued,
			StartDate:    "2024-09-01",
			EndDate:      "2024-11-30",
			RepoTemplate: "git@gitlab.local/mlops-template.git",
			Description:  "Продвинутые практики MLOps",
			URL:          "/course/mlops",
		},
		"rust": {
			ID:           "rust",
			Name:         "Rust Core",
			Status:       CourseStatusCreated,
			StartDate:    "2024-10-15",
			EndDate:      "2025-01-15",
			RepoTemplate: "git@gitlab.local/rust-template.git",
			Description:  "Основы системного программирования на Rust",
			URL:          "/course/rust",
		},
		"golang": {
			ID:           "golang",
			Name:         "Go Lab",
			Status:       CourseStatusFinished,
			StartDate:    "2024-08-01",
			EndDate:      "2024-10-31",
			RepoTemplate: "git@gitlab.local/golang-template.git",
			Description:  "Практикум по языку Go",
			URL:          "/course/golang",
		},
		"advanced-cpp": {
			ID:           "advanced-cpp",
			Name:         "Advanced C++",
			Status:       CourseStatusInProgress,
			StartDate:    "2024-10-01",
			EndDate:      "2024-12-20",
			RepoTemplate: "git@gitlab.local/advanced-cpp-template.git",
			Description:  "Продвинутые концепции C++",
			URL:          "/course/advanced-cpp",
		},
		"advanced-python": {
			ID:           "advanced-python",
			Name:         "Advanced Python",
			Status:       CourseStatusCreated,
			StartDate:    "2024-11-01",
			EndDate:      "2025-02-28",
			RepoTemplate: "git@gitlab.local/advanced-python-template.git",
			Description:  "Продвинутый анализ данных на Python",
			URL:          "/course/advanced-python",
		},
	}

	courseMu sync.RWMutex
)

// Валидация статуса курса
func isValidCourseStatus(status string) bool {
	validStatuses := []string{
		string(CourseStatusCreated),
		string(CourseStatusHidden),
		string(CourseStatusInProgress),
		string(CourseStatusAllTasksIssued),
		string(CourseStatusDoreshka),
		string(CourseStatusFinished),
	}

	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// Валидация дат
func isValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func isValidSlug(slug string) bool {
	if len(slug) < MinSlugLength || len(slug) > MaxSlugLength {
		return false
	}
	if len(slug) == 0 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	match, _ := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*$`, slug)
	return match
}

// Проверка логики дат
func isValidDateRange(start, end string) bool {
	startDate, err1 := time.Parse("2006-01-02", start)
	endDate, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil {
		return false
	}
	return endDate.After(startDate)
}

// Валидация токена
func isValidToken(token string) bool {
	if len(token) < MinTokenLength || len(token) > MaxTokenLength {
		return false
	}
	match, _ := regexp.MatchString(`^[A-Za-z0-9._-]+$`, token)
	return match
}

// Validate - универсальная валидация запроса для создания курса
func (req *PostCourseRequest) Validate() []ValidationError {
	var errors []ValidationError

	if req.Name == "" {
		errors = append(errors, ValidationError{"name", "name is required"})
	} else if len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength {
		errors = append(errors, ValidationError{"name", "name must be between 3 and 100 characters"})
	}

	if req.Slug == "" {
		errors = append(errors, ValidationError{"slug", "slug is required"})
	} else if !isValidSlug(req.Slug) {
		errors = append(errors, ValidationError{"slug", "slug must contain only lowercase letters, numbers and hyphens (3-50 chars)"})
	}

	if req.Status == "" {
		errors = append(errors, ValidationError{"status", "status is required"})
	} else if !isValidCourseStatus(req.Status) {
		errors = append(errors, ValidationError{"status", "invalid status value. Must be one of: created, hidden, in_progress, all_tasks_issued, doreshka, finished"})
	}

	if req.StartDate == "" {
		errors = append(errors, ValidationError{"startDate", "startDate is required"})
	} else if !isValidDate(req.StartDate) {
		errors = append(errors, ValidationError{"startDate", "startDate must be in format YYYY-MM-DD"})
	}

	if req.EndDate == "" {
		errors = append(errors, ValidationError{"endDate", "endDate is required"})
	} else if !isValidDate(req.EndDate) {
		errors = append(errors, ValidationError{"endDate", "endDate must be in format YYYY-MM-DD"})
	}

	if req.StartDate != "" && req.EndDate != "" && !isValidDateRange(req.StartDate, req.EndDate) {
		errors = append(errors, ValidationError{"dateRange", "endDate must be after startDate"})
	}

	if req.RepoTemplate == "" {
		errors = append(errors, ValidationError{"repoTemplate", "repoTemplate is required"})
	}

	if req.Description == "" {
		errors = append(errors, ValidationError{"description", "description is required"})
	} else if len(req.Description) > MaxDescLength {
		errors = append(errors, ValidationError{"description", "description must not exceed 500 characters"})
	}

	return errors
}

// AuthMiddleware проверяет наличие и валидность токена авторизации
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")

		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "authorization header is required",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid authorization header format",
			})
		}

		token := strings.TrimSpace(parts[1])

		if len(token) < MinTokenLength || len(token) > MaxTokenLength {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid token length",
			})
		}

		if !isValidToken(token) {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid token format",
			})
		}

		c.Set("token", token)
		return next(c)
	}
}

// GET `/api/courses` - Получить все курсы
func GetCoursesHandler(c echo.Context) error {
	statusFilter := c.QueryParam("status")

	courseMu.RLock()
	defer courseMu.RUnlock()

	courses := make([]Course, 0, len(courseDB))
	for _, course := range courseDB {
		if statusFilter == "" || string(course.Status) == statusFilter {
			courses = append(courses, course)
		}
	}

	return c.JSON(http.StatusOK, courses)
}

// GET `/api/courses/:courseId` - Получить курс по ID (slug)
func GetCourseHandler(c echo.Context) error {
	courseID := c.Param("courseId")

	courseMu.RLock()
	course, exists := courseDB[courseID]
	courseMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "course not found",
		})
	}

	return c.JSON(http.StatusOK, course)
}

// POST `/api/courses` - Создать новый курс
func CreateCourseHandler(c echo.Context) error {
	var req PostCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid JSON payload",
		})
	}

	// Используем унифицированную валидацию
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "validation failed",
			"details": validationErrors,
		})
	}

	// Проверка существования курса
	courseMu.RLock()
	_, exists := courseDB[req.Slug]
	courseMu.RUnlock()

	if exists {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "course with this slug already exists",
		})
	}

	course := Course{
		ID:           req.Slug,
		Name:         req.Name,
		Status:       CourseStatus(req.Status),
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		RepoTemplate: req.RepoTemplate,
		Description:  req.Description,
		URL:          "/course/" + req.Slug,
	}

	courseMu.Lock()
	courseDB[req.Slug] = course
	courseMu.Unlock()

	return c.JSON(http.StatusCreated, course)
}

// PUT `/api/courses/:courseId` - Обновить курс
func UpdateCourseHandler(c echo.Context) error {
	courseID := c.Param("courseId")

	courseMu.RLock()
	course, exists := courseDB[courseID]
	courseMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "course not found",
		})
	}

	var req PostCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid JSON payload",
		})
	}

	// Частичная валидация для PUT (проверяем только переданные поля)
	if req.Name != "" && (len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "name must be between 3 and 100 characters",
		})
	}

	if req.Status != "" && !isValidCourseStatus(req.Status) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid status value. Must be one of: created, hidden, in_progress, all_tasks_issued, doreshka, finished",
		})
	}

	if req.StartDate != "" && !isValidDate(req.StartDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "startDate must be in format YYYY-MM-DD",
		})
	}

	if req.EndDate != "" && !isValidDate(req.EndDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "endDate must be in format YYYY-MM-DD",
		})
	}

	if req.Description != "" && len(req.Description) > MaxDescLength {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description must not exceed 500 characters",
		})
	}

	// Создаем обновленный курс
	updatedCourse := course

	if req.Name != "" {
		updatedCourse.Name = req.Name
	}
	if req.Status != "" {
		updatedCourse.Status = CourseStatus(req.Status)
	}
	if req.StartDate != "" {
		updatedCourse.StartDate = req.StartDate
	}
	if req.EndDate != "" {
		updatedCourse.EndDate = req.EndDate
	}
	if req.RepoTemplate != "" {
		updatedCourse.RepoTemplate = req.RepoTemplate
	}
	if req.Description != "" {
		updatedCourse.Description = req.Description
	}

	// Проверка диапазона дат после всех обновлений
	if !isValidDateRange(updatedCourse.StartDate, updatedCourse.EndDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "endDate must be after startDate",
		})
	}

	courseMu.Lock()
	courseDB[courseID] = updatedCourse
	courseMu.Unlock()

	return c.JSON(http.StatusOK, updatedCourse)
}