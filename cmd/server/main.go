package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/fidelis27/secretaria-backend/internal/adapters/httpserver"
	"github.com/fidelis27/secretaria-backend/internal/adapters/mariadb"
	applicationenrollment "github.com/fidelis27/secretaria-backend/internal/application/enrollment"
	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
	applicationgroup "github.com/fidelis27/secretaria-backend/internal/application/group"
	"github.com/fidelis27/secretaria-backend/internal/application/health"
	applicationinstitution "github.com/fidelis27/secretaria-backend/internal/application/institution"
	applicationstudent "github.com/fidelis27/secretaria-backend/internal/application/student"
	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	"github.com/fidelis27/secretaria-backend/internal/domain/authorization"
	"github.com/fidelis27/secretaria-backend/internal/platform/config"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	database, err := mariadb.Open(appConfig)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	address := fmt.Sprintf("0.0.0.0:%d", appConfig.Port)
	institutionRepository := mariadb.NewInstitutionRepository(database)
	institutionService := applicationinstitution.NewService(institutionRepository)
	userRepository := mariadb.NewUserRepository(database)
	userService := applicationuser.NewService(userRepository)
	studentRepository := mariadb.NewStudentRepository(database)
	studentService := applicationstudent.NewService(studentRepository)
	enrollmentRepository := mariadb.NewEnrollmentRepository(database)
	enrollmentService := applicationenrollment.NewService(enrollmentRepository)
	eventBus := applicationevent.NewBus()
	eventRepository := mariadb.NewEventRepository(database)
	eventService := applicationevent.NewService(eventRepository, eventBus)
	membershipRepository := mariadb.NewMemberGroupRepository(database)
	policy := authorization.NewPolicy(membershipRepository)
	groupRepository := mariadb.NewGroupRepository(database)
	groupService := applicationgroup.NewService(groupRepository)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	server := httpserver.New(address, health.NewService(database), institutionService, userService, studentService, enrollmentService, eventService, eventBus, groupService, policy)
	slog.Info("server listening", "address", address)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
