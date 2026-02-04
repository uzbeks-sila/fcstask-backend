package handler

import (
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	MinSlugLength  = 3
	MaxSlugLength  = 50
	MinNameLength  = 3
	MaxNameLength  = 100
	MaxDescLength  = 500
)

// CourseStatus - статус курса
type CourseStatus string

const (
	CourseStatusCreated        CourseStatus = "created"
	CourseStatusHidden         CourseStatus = "hidden"
	CourseStatusInProgress     CourseStatus = "in_progress"
	CourseStatusAllTasksIssued CourseStatus = "all_tasks_issued"
	CourseStatusDoreshka       CourseStatus = "doreshka"
	CourseStatusFinished       CourseStatus = "finished"
)

// Course - модель курса
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

// PostCourseRequest - тело запроса на создание курса
type PostCourseRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	RepoTemplate string `json:"repoTemplate"`
	Description  string `json:"description"`
}

// ValidationError - ошибка валидации
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// In-memory storage
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

// Вспомогательные функции валидации

func isValidCourseStatus(status string) bool {
	valid := map[string]bool{
		string(CourseStatusCreated):        true,
		string(CourseStatusHidden):         true,
		string(CourseStatusInProgress):     true,
		string(CourseStatusAllTasksIssued): true,
		string(CourseStatusDoreshka):       true,
		string(CourseStatusFinished):       true,
	}
	return valid[status]
}

func isValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func isValidSlug(slug string) bool {
	if len(slug) < MinSlugLength || len(slug) > MaxSlugLength {
		return false
	}
	if slug == "" || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	match, _ := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*$`, slug)
	return match
}

func isValidDateRange(start, end string) bool {
	startDate, _ := time.Parse("2006-01-02", start)
	endDate, _ := time.Parse("2006-01-02", end)
	return endDate.After(startDate)
}

// Validate проверяет корректность запроса
func (req *PostCourseRequest) Validate() []ValidationError {
	var errs []ValidationError

	if req.Name == "" {
		errs = append(errs, ValidationError{"name", "name is required"})
	} else if len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength {
		errs = append(errs, ValidationError{"name", "name must be between 3 and 100 characters"})
	}

	if req.Slug == "" {
		errs = append(errs, ValidationError{"slug", "slug is required"})
	} else if !isValidSlug(req.Slug) {
		errs = append(errs, ValidationError{"slug", "slug must contain only lowercase letters, numbers and hyphens (3-50 chars)"})
	}

	if req.Status == "" {
		errs = append(errs, ValidationError{"status", "status is required"})
	} else if !isValidCourseStatus(req.Status) {
		errs = append(errs, ValidationError{"status", "invalid status value"})
	}

	if req.StartDate == "" {
		errs = append(errs, ValidationError{"startDate", "startDate is required"})
	} else if !isValidDate(req.StartDate) {
		errs = append(errs, ValidationError{"startDate", "startDate must be in format YYYY-MM-DD"})
	}

	if req.EndDate == "" {
		errs = append(errs, ValidationError{"endDate", "endDate is required"})
	} else if !isValidDate(req.EndDate) {
		errs = append(errs, ValidationError{"endDate", "endDate must be in format YYYY-MM-DD"})
	}

	if req.StartDate != "" && req.EndDate != "" && !isValidDateRange(req.StartDate, req.EndDate) {
		errs = append(errs, ValidationError{"dateRange", "endDate must be after startDate"})
	}

	if req.RepoTemplate == "" {
		errs = append(errs, ValidationError{"repoTemplate", "repoTemplate is required"})
	}

	if req.Description == "" {
		errs = append(errs, ValidationError{"description", "description is required"})
	} else if len(req.Description) > MaxDescLength {
		errs = append(errs, ValidationError{"description", "description must not exceed 500 characters"})
	}

	return errs
}

// Хендлеры

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

func GetCourseHandler(c echo.Context) error {
	courseID := c.Param("courseId")

	courseMu.RLock()
	course, exists := courseDB[courseID]
	courseMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
	}

	return c.JSON(http.StatusOK, course)
}

func CreateCourseHandler(c echo.Context) error {
	var req PostCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
	}

	if errs := req.Validate(); len(errs) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "validation failed", "details": errs})
	}

	courseMu.RLock()
	_, exists := courseDB[req.Slug]
	courseMu.RUnlock()

	if exists {
		return c.JSON(http.StatusConflict, map[string]string{"error": "course with this slug already exists"})
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

func UpdateCourseHandler(c echo.Context) error {
	courseID := c.Param("courseId")

	courseMu.RLock()
	course, exists := courseDB[courseID]
	courseMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "course not found"})
	}

	var req PostCourseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
	}

	// Валидация только непустых полей
	if req.Name != "" && (len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name must be between 3 and 100 characters"})
	}
	if req.Status != "" && !isValidCourseStatus(req.Status) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid status value"})
	}
	if req.StartDate != "" && !isValidDate(req.StartDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "startDate must be in format YYYY-MM-DD"})
	}
	if req.EndDate != "" && !isValidDate(req.EndDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "endDate must be in format YYYY-MM-DD"})
	}
	if req.Description != "" && len(req.Description) > MaxDescLength {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "description must not exceed 500 characters"})
	}

	updated := course
	if req.Name != "" {
		updated.Name = req.Name
	}
	if req.Status != "" {
		updated.Status = CourseStatus(req.Status)
	}
	if req.StartDate != "" {
		updated.StartDate = req.StartDate
	}
	if req.EndDate != "" {
		updated.EndDate = req.EndDate
	}
	if req.RepoTemplate != "" {
		updated.RepoTemplate = req.RepoTemplate
	}
	if req.Description != "" {
		updated.Description = req.Description
	}

	if !isValidDateRange(updated.StartDate, updated.EndDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "endDate must be after startDate"})
	}

	courseMu.Lock()
	courseDB[courseID] = updated
	courseMu.Unlock()

	return c.JSON(http.StatusOK, updated)
}