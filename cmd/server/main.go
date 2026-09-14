package main

import (
	"fmt"
	"log"

	"github.com/karol/secretaria-escolar-backend/internal/adapters/httpserver"
	"github.com/karol/secretaria-escolar-backend/internal/adapters/mariadb"
	"github.com/karol/secretaria-escolar-backend/internal/application/health"
	applicationinstitution "github.com/karol/secretaria-escolar-backend/internal/application/institution"
	applicationstudent "github.com/karol/secretaria-escolar-backend/internal/application/student"
	applicationuser "github.com/karol/secretaria-escolar-backend/internal/application/user"
	"github.com/karol/secretaria-escolar-backend/internal/platform/config"
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
	server := httpserver.New(address, health.NewService(database), institutionService, userService, studentService)
	log.Printf("server listening on %s", address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
