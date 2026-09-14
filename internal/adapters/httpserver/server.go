package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	applicationenrollment "github.com/fidelis27/secretaria-backend/internal/application/enrollment"
	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	"github.com/fidelis27/secretaria-backend/internal/application/health"
	applicationinstitution "github.com/fidelis27/secretaria-backend/internal/application/institution"
	applicationstudent "github.com/fidelis27/secretaria-backend/internal/application/student"
	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	domaininstitution "github.com/fidelis27/secretaria-backend/internal/domain/institution"
	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string, healthService health.Service, institutionService applicationinstitution.Service, userService applicationuser.Service, studentService applicationstudent.Service, enrollmentService applicationenrollment.Service, eventService applicationevent.Service, eventBus *applicationevent.Bus) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		status := healthService.Check(request.Context())
		if !status.OK {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(writer).Encode(status)
	})
	mux.HandleFunc("GET /institutions", func(writer http.ResponseWriter, request *http.Request) {
		institutions, err := institutionService.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list institutions")
			return
		}
		writeJSON(writer, http.StatusOK, institutions)
	})
	mux.HandleFunc("POST /institutions", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Name string `json:"name"`
			CNPJ string `json:"cnpj"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := institutionService.Create(request.Context(), input.Name, input.CNPJ)
		if errors.Is(err, domaininstitution.ErrNameRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create institution")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("GET /users", func(writer http.ResponseWriter, request *http.Request) {
		users, err := userService.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list users")
			return
		}
		writeJSON(writer, http.StatusOK, users)
	})
	mux.HandleFunc("POST /users", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Name       string `json:"name"`
			Email      string `json:"email"`
			SuperAdmin bool   `json:"superAdmin"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := userService.Create(request.Context(), input.Name, input.Email, input.SuperAdmin)
		if errors.Is(err, domainuser.ErrNameRequired) || errors.Is(err, domainuser.ErrEmailRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create user")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("GET /students", func(writer http.ResponseWriter, request *http.Request) {
		students, err := studentService.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list students")
			return
		}
		writeJSON(writer, http.StatusOK, students)
	})
	mux.HandleFunc("POST /students", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Name          string `json:"name"`
			InstitutionID string `json:"institutionId"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := studentService.Create(request.Context(), input.Name, input.InstitutionID)
		if errors.Is(err, domainstudent.ErrNameRequired) || errors.Is(err, domainstudent.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create student")
			return
		}
		domainEvent, err := domainevent.New("STUDENT_CREATED", "backend.student", request.Header.Get("x-correlation-id"), created)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create student event")
			return
		}
		if err := eventService.Publish(request.Context(), domainEvent); err != nil {
			writeError(writer, http.StatusInternalServerError, "could not publish student event")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.Handle("GET /events", eventHandler(eventBus))
	mux.HandleFunc("GET /enrollments", func(writer http.ResponseWriter, request *http.Request) {
		enrollments, err := enrollmentService.List(request.Context())
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list enrollments")
			return
		}
		writeJSON(writer, http.StatusOK, enrollments)
	})
	mux.HandleFunc("POST /enrollments", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			StudentID     string `json:"studentId"`
			InstitutionID string `json:"institutionId"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := enrollmentService.Create(request.Context(), input.StudentID, input.InstitutionID)
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create enrollment")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/transfer", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			InstitutionID string `json:"institutionId"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := enrollmentService.Transfer(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"), input.InstitutionID)
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrEnrollmentIDRequired) || errors.Is(err, domainenrollment.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domainenrollment.ErrSourceEnrollmentNotActive) {
			writeError(writer, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not transfer enrollment")
			return
		}
		domainEvent, err := domainevent.New("STUDENT_TRANSFERRED", "backend.enrollment", request.Header.Get("x-correlation-id"), map[string]string{
			"studentId":             request.PathValue("studentId"),
			"sourceEnrollment":      request.PathValue("enrollmentId"),
			"destinationEnrollment": created.ID,
			"institutionId":         created.InstitutionID,
		})
		if err != nil || eventService.Publish(request.Context(), domainEvent) != nil {
			writeError(writer, http.StatusInternalServerError, "could not publish transfer event")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/suspend", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Reason string `json:"reason"`
			Date   string `json:"date"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		suspendedAt, err := time.Parse(time.RFC3339, input.Date)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "date must be RFC3339")
			return
		}
		err = enrollmentService.Suspend(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"), input.Reason, suspendedAt)
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrEnrollmentIDRequired) || errors.Is(err, domainenrollment.ErrSuspensionReasonRequired) || errors.Is(err, domainenrollment.ErrSuspensionDateRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domainenrollment.ErrEnrollmentNotActive) {
			writeError(writer, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not suspend enrollment")
			return
		}
		domainEvent, err := domainevent.New("ENROLLMENT_SUSPENDED", "backend.enrollment", request.Header.Get("x-correlation-id"), map[string]string{
			"studentId":    request.PathValue("studentId"),
			"enrollmentId": request.PathValue("enrollmentId"),
			"reason":       input.Reason,
			"date":         input.Date,
		})
		if err != nil || eventService.Publish(request.Context(), domainEvent) != nil {
			writeError(writer, http.StatusInternalServerError, "could not publish suspension event")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "suspended"})
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/reopen", func(writer http.ResponseWriter, request *http.Request) {
		err := enrollmentService.Reopen(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"))
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrEnrollmentIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domainenrollment.ErrEnrollmentNotSuspended) {
			writeError(writer, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not reopen enrollment")
			return
		}
		domainEvent, err := domainevent.New("ENROLLMENT_REOPENED", "backend.enrollment", request.Header.Get("x-correlation-id"), map[string]string{
			"studentId":    request.PathValue("studentId"),
			"enrollmentId": request.PathValue("enrollmentId"),
		})
		if err != nil || eventService.Publish(request.Context(), domainEvent) != nil {
			writeError(writer, http.StatusInternalServerError, "could not publish reopening event")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "active"})
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: identityMiddleware(mux, userService),
		},
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func (server *Server) ListenAndServe() error {
	return server.httpServer.ListenAndServe()
}
