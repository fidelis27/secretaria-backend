package main

import (
	"fmt"
	"log"

	"github.com/fidelis27/secretaria-backend/internal/adapters/httpserver"
	"github.com/fidelis27/secretaria-backend/internal/adapters/mariadb"
	applicationenrollment "github.com/fidelis27/secretaria-backend/internal/application/enrollment"
	applicationevent "github.com/fidelis27/secretaria-backend/internal/application/event"
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
		log.Fatal(err)
	}
	database, err := mariadb.Open(appConfig)
	if err != nil {
		log.Fatal(err)
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
	server := httpserver.New(address, health.NewService(database), institutionService, userService, studentService, enrollmentService, eventService, eventBus, policy)
	log.Printf("server listening on %s", address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
