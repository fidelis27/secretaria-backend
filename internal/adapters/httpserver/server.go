package httpserver

import (
	"encoding/json"
	"errors"
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
	domainevent "github.com/fidelis27/secretaria-backend/internal/domain/event"
	domaingroup "github.com/fidelis27/secretaria-backend/internal/domain/group"
	domaininstitution "github.com/fidelis27/secretaria-backend/internal/domain/institution"
	domainstudent "github.com/fidelis27/secretaria-backend/internal/domain/student"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string, healthService health.Service, institutionService applicationinstitution.Service, userService applicationuser.Service, studentService applicationstudent.Service, enrollmentService applicationenrollment.Service, eventService applicationevent.Service, eventBus *applicationevent.Bus, groupService applicationgroup.Service, policy domainauthorization.Policy) *Server {
	mux := http.NewServeMux()
	metrics := newRequestMetrics()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		status := healthService.Check(request.Context())
		if !status.OK {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(writer).Encode(status)
	})
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
			writeError(writer, http.StatusInternalServerError, "could not create institution")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.Handle("GET /users", userListHandler(userService, policy))
	mux.Handle("POST /users", userCreateHandler(userService, policy))
	mux.HandleFunc("GET /groups", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve group scope")
			return
		}
		var groups []domaingroup.Group
		if user.SuperAdmin {
			groups, err = groupService.List(request.Context())
		} else {
			groups, err = groupService.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list groups")
			return
		}
		writeJSON(writer, http.StatusOK, groups)
	})
	mux.HandleFunc("POST /groups", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		if !policy.CanCreateGroup(user) {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		var input struct {
			InstitutionID string `json:"institutionId"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := groupService.Create(request.Context(), input.InstitutionID)
		if errors.Is(err, domaingroup.ErrInstitutionIDRequired) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create group")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("GET /groups/{groupId}/members", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		group, found, err := groupService.FindByID(request.Context(), request.PathValue("groupId"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load group")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "group not found")
			return
		}
		allowed, err := policy.CanManageGroup(request.Context(), user, group.ID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not verify group authorization")
			return
		}
		if !allowed {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		memberships, err := groupService.ListMemberships(request.Context(), group.ID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not list memberships")
			return
		}
		writeJSON(writer, http.StatusOK, memberships)
	})
	mux.HandleFunc("POST /groups/{groupId}/members", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		group, found, err := groupService.FindByID(request.Context(), request.PathValue("groupId"))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load group")
			return
		}
		if !found {
			writeError(writer, http.StatusNotFound, "group not found")
			return
		}
		allowed, err := policy.CanManageGroup(request.Context(), user, group.ID)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not verify group authorization")
			return
		}
		if !allowed {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		var input struct {
			UserID string                   `json:"userId"`
			Role   domainauthorization.Role `json:"role"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		created, err := groupService.AddMembership(request.Context(), input.UserID, group.ID, group.InstitutionID, input.Role)
		if errors.Is(err, domaingroup.ErrUserIDRequired) || errors.Is(err, domaingroup.ErrGroupIDRequired) || errors.Is(err, domaingroup.ErrInvalidRole) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not create membership")
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	})
	mux.HandleFunc("GET /students", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve institution scope")
			return
		}
		var students []domainstudent.Student
		if user.SuperAdmin {
			students, err = studentService.List(request.Context())
		} else {
			students, err = studentService.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
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
		if !authorizeInstitutionEdit(writer, request, policy, input.InstitutionID) {
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
		domainEvent, err := domainevent.NewForInstitution("STUDENT_CREATED", "backend.student", request.Header.Get("x-correlation-id"), created.InstitutionID, created)
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
	mux.Handle("GET /events", eventHandler(eventBus, policy))
	mux.HandleFunc("GET /metrics", metrics.handler)
	mux.Handle("GET /events/history", eventHistoryHandler(eventService, policy))
	mux.HandleFunc("GET /enrollments", func(writer http.ResponseWriter, request *http.Request) {
		user, _ := userFromContext(request.Context())
		institutionIDs, err := policy.VisibleInstitutionIDs(request.Context(), user)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not resolve institution scope")
			return
		}
		var enrollments []domainenrollment.Enrollment
		if user.SuperAdmin {
			enrollments, err = enrollmentService.List(request.Context())
		} else {
			enrollments, err = enrollmentService.ListByInstitutionIDs(request.Context(), institutionIDs)
		}
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
		if !authorizeInstitutionEdit(writer, request, policy, input.InstitutionID) {
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
			writeError(writer, http.StatusInternalServerError, "could not transfer enrollment")
			return
		}
		domainEvent, err := domainevent.NewForInstitutions("STUDENT_TRANSFERRED", "backend.enrollment", request.Header.Get("x-correlation-id"), []string{existing.InstitutionID, created.InstitutionID}, map[string]string{
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
			writeError(writer, http.StatusInternalServerError, "could not suspend enrollment")
			return
		}
		domainEvent, err := domainevent.NewForInstitution("ENROLLMENT_SUSPENDED", "backend.enrollment", request.Header.Get("x-correlation-id"), existing.InstitutionID, map[string]string{
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
			writeError(writer, http.StatusInternalServerError, "could not reopen enrollment")
			return
		}
		domainEvent, err := domainevent.NewForInstitution("ENROLLMENT_REOPENED", "backend.enrollment", request.Header.Get("x-correlation-id"), existing.InstitutionID, map[string]string{
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
			Handler: observabilityMiddleware(corsMiddleware(identityMiddleware(mux, userService)), slog.Default(), metrics),
		},
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if isLocalFrontendOrigin(origin) {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
			writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-demo-user, x-correlation-id")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func isLocalFrontendOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	configuredOrigins := os.Getenv("CORS_ORIGINS")
	if configuredOrigins == "" {
		configuredOrigins = "http://localhost:4173,http://localhost:4174,http://localhost:4175,http://localhost:4176,http://localhost:4178,http://localhost:4179,http://127.0.0.1:4173,http://127.0.0.1:4174,http://127.0.0.1:4175,http://127.0.0.1:4176,http://127.0.0.1:4178,http://127.0.0.1:4179"
	}
	for _, configuredOrigin := range strings.Split(configuredOrigins, ",") {
		if strings.TrimSpace(configuredOrigin) == origin {
			return true
		}
	}
	return false
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

func (server *Server) ListenAndServe() error {
	return server.httpServer.ListenAndServe()
}
