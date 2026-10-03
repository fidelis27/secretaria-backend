package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	applicationenrollment "github.com/fidelis27/secretaria-backend/internal/application/enrollment"
	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	applicationgroup "github.com/fidelis27/secretaria-backend/internal/application/group"
	"github.com/fidelis27/secretaria-backend/internal/application/health"
	applicationinstitution "github.com/fidelis27/secretaria-backend/internal/application/institution"
	applicationstudent "github.com/fidelis27/secretaria-backend/internal/application/student"
	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainauthorization "github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	domainenrollment "github.com/fidelis27/secretaria-backend/internal/domain/enrollment"
	domaininstitution "github.com/fidelis27/secretaria-backend/internal/domain/institution"
	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
	apperrors "github.com/fidelis27/secretaria-backend/internal/platform/errors"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string, healthService health.Service, institutionService applicationinstitution.Service, userService applicationuser.Service, studentService applicationstudent.Service, enrollmentService applicationenrollment.Service, eventService applicationevent.Service, eventBus *applicationevent.Bus, groupService applicationgroup.Service, policy domainauthorization.Policy, verifier identityTokenVerifier) *Server {
	mux := http.NewServeMux()
	metrics := newRequestMetrics()
	mux.Handle("GET /health", healthHandler(healthService))
	mux.HandleFunc("GET /institutions", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve institution scope")
			return
		}
		var institutions []domaininstitution.Institution
		if user.SuperAdmin {
			institutions, err = institutionService.List(request.Context())
		} else {
			institutions, err = institutionService.ListByIDs(request.Context(), institutionIDs)
		}
		if err != nil {
			writeServiceError(writer, err, "could not list institutions")
			return
		}
		writeJSON(writer, http.StatusOK, institutions)
	})
	mux.HandleFunc("POST /institutions", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Name string `json:"name"`
			CNPJ string `json:"cnpj"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		user, _ := userFromContext(request.Context())
		if !user.SuperAdmin {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		created, err := institutionService.Create(request.Context(), input.Name, input.CNPJ)
		if errors.Is(err, domaininstitution.ErrNameRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create institution")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.Handle("GET /users", userListHandler(userService, policy))
	mux.Handle("POST /users", userCreateHandler(userService, policy))
	mux.Handle("PATCH /users/{userId}", userUpdateHandler(userService, policy, eventService))
	mux.Handle("GET /groups", groupListHandler(groupService, policy))
	mux.Handle("POST /groups", groupCreateHandler(groupService, policy))
	mux.Handle("GET /groups/{groupId}/members", groupMembershipListHandler(groupService, policy))
	mux.Handle("POST /groups/{groupId}/members", groupMembershipCreateHandler(groupService, policy))
	mux.Handle("PATCH /groups/{groupId}/members/{userId}", groupMembershipUpdateHandler(groupService, policy, eventService))
	mux.Handle("DELETE /groups/{groupId}/members/{userId}", groupMembershipDeleteHandler(groupService, policy, eventService))
	mux.Handle("GET /groups/{groupId}/candidates", groupCandidatesHandler(groupService, policy))
	mux.HandleFunc("GET /students", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeServiceError(writer, err, "could not resolve institution scope")
			return
		}
		var students []domainstudent.Student
		if user.SuperAdmin {
			students, err = studentService.List(request.Context())
		} else {
			students, err = studentService.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
		if err != nil {
			writeServiceError(writer, err, "could not list students")
			return
		}
		writeJSON(writer, http.StatusOK, students)
	})
	mux.HandleFunc("POST /students", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Name          string `json:"name"`
			InstitutionID string `json:"institutionId"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if !authorizeInstitutionEdit(writer, request, policy, input.InstitutionID) {
			return
		}
		created, err := studentService.Create(request.Context(), input.Name, input.InstitutionID)
		if errors.Is(err, domainstudent.ErrNameRequired) || errors.Is(err, domainstudent.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create student")
			return
		}
		publishDomainEventBestEffort(request.Context(), eventService, "STUDENT_CREATED", "backend.student", request.Header.Get("x-correlation-id"), []string{created.InstitutionID}, created)
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.Handle("GET /events", eventHandler(eventBus, policy))
	mux.Handle("GET /metrics", metricsHandler(metrics))
	mux.Handle("GET /events/history", eventHistoryHandler(eventService, policy))
	mux.HandleFunc("GET /enrollments", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeServiceError(writer, err, "could not resolve institution scope")
			return
		}
		var enrollments []domainenrollment.Enrollment
		if user.SuperAdmin {
			enrollments, err = enrollmentService.List(request.Context())
		} else {
			enrollments, err = enrollmentService.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
		if err != nil {
			writeServiceError(writer, err, "could not list enrollments")
			return
		}
		writeJSON(writer, http.StatusOK, enrollments)
	})
	mux.HandleFunc("POST /enrollments", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			StudentID     string `json:"studentId"`
			InstitutionID string `json:"institutionId"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if !authorizeInstitutionEdit(writer, request, policy, input.InstitutionID) {
			return
		}
		created, err := enrollmentService.Create(request.Context(), input.StudentID, input.InstitutionID)
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not create enrollment")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/transfer", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			InstitutionID string `json:"institutionId"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		existing, found, err := enrollmentService.FindByID(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load enrollment")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "enrollment not found")
			return
		}
		if !authorizeInstitutionEdit(writer, request, policy, existing.InstitutionID) || !authorizeInstitutionEdit(writer, request, policy, input.InstitutionID) {
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
			writeServiceError(writer, err, "could not transfer enrollment")
			return
		}
		publishDomainEventBestEffort(request.Context(), eventService, "STUDENT_TRANSFERRED", "backend.enrollment", request.Header.Get("x-correlation-id"), []string{existing.InstitutionID, created.InstitutionID}, map[string]string{
			"studentId":             request.PathValue("studentId"),
			"sourceEnrollment":      request.PathValue("enrollmentId"),
			"destinationEnrollment": created.ID,
			"institutionId":         created.InstitutionID,
		})
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/suspend", func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Reason string `json:"reason"`
			Date   string `json:"date"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		suspendedAt, err := time.Parse(time.RFC3339, input.Date)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "date must be RFC3339")
			return
		}
		existing, found, err := enrollmentService.FindByID(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load enrollment")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "enrollment not found")
			return
		}
		if !authorizeInstitutionEdit(writer, request, policy, existing.InstitutionID) {
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
			writeServiceError(writer, err, "could not suspend enrollment")
			return
		}
		publishDomainEventBestEffort(request.Context(), eventService, "ENROLLMENT_SUSPENDED", "backend.enrollment", request.Header.Get("x-correlation-id"), []string{existing.InstitutionID}, map[string]string{
			"studentId":    request.PathValue("studentId"),
			"enrollmentId": request.PathValue("enrollmentId"),
			"reason":       input.Reason,
			"date":         input.Date,
		})
		writeJSON(writer, http.StatusOK, map[string]string{"status": "suspended"})
	})
	mux.HandleFunc("POST /students/{studentId}/enrollments/{enrollmentId}/reopen", func(writer http.ResponseWriter, request *http.Request) {
		existing, found, err := enrollmentService.FindByID(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load enrollment")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "enrollment not found")
			return
		}
		if !authorizeInstitutionEdit(writer, request, policy, existing.InstitutionID) {
			return
		}
		err = enrollmentService.Reopen(request.Context(), request.PathValue("studentId"), request.PathValue("enrollmentId"))
		if errors.Is(err, domainenrollment.ErrStudentIDRequired) || errors.Is(err, domainenrollment.ErrEnrollmentIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domainenrollment.ErrEnrollmentNotSuspended) {
			writeError(writer, http.StatusConflict, err.Error())
			return
		}
		if err != nil {
			writeServiceError(writer, err, "could not reopen enrollment")
			return
		}
		publishDomainEventBestEffort(request.Context(), eventService, "ENROLLMENT_REOPENED", "backend.enrollment", request.Header.Get("x-correlation-id"), []string{existing.InstitutionID}, map[string]string{
			"studentId":    request.PathValue("studentId"),
			"enrollmentId": request.PathValue("enrollmentId"),
		})
		writeJSON(writer, http.StatusOK, map[string]string{"status": "active"})
	})

	return &Server{
		httpServer: newHTTPServer(addr, observabilityMiddleware(corsMiddleware(identityMiddleware(mux, userService, verifier)), slog.Default(), metrics)),
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if isLocalFrontendOrigin(origin) {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
			writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Sec-WebSocket-Protocol, x-correlation-id")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func healthHandler(healthService health.Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		status := healthService.Check(request.Context())
		if !status.OK {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(writer).Encode(status)
	})
}

func isLocalFrontendOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, configuredOrigin := range strings.Split(os.Getenv("CORS_ORIGINS"), ",") {
		if strings.TrimSpace(configuredOrigin) == origin {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		return false
	}
	return isLoopbackOrigin(origin)
}

func isLoopbackOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:")
}

func authorizeInstitutionEdit(writer http.ResponseWriter, request *http.Request, policy domainauthorization.Policy, institutionID string) bool {
	user, ok := userFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return false
	}
	allowed, err := policy.CanEditInstitution(request.Context(), user, institutionID)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not verify authorization")
		return false
	}
	if !allowed {
		writeError(writer, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON body")
	}
	return nil
}

func writeServiceError(writer http.ResponseWriter, err error, fallback string) {
	if apperrors.IsDatabaseUnavailable(err) {
		writeError(writer, http.StatusFailedDependency, apperrors.ErrDatabaseUnavailable.Error())
		return
	}
	if apperrors.IsDuplicateEntry(err) {
		writeError(writer, http.StatusConflict, apperrors.DuplicateEntryMessage(err))
		return
	}
	if apperrors.IsForeignKeyViolation(err) {
		writeError(writer, http.StatusUnprocessableEntity, apperrors.ErrForeignKeyViolation.Error())
		return
	}
	writeError(writer, http.StatusInternalServerError, fallback)
}

func (server *Server) ListenAndServe() error {
	return server.httpServer.ListenAndServe()
}
