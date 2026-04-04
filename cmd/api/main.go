package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jlucaspains/sharp-cred-manager/internal/handlers"
	"github.com/jlucaspains/sharp-cred-manager/internal/jobs"
	"github.com/jlucaspains/sharp-cred-manager/internal/midlewares"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
	"github.com/joho/godotenv"
)

var checkCertJob = &jobs.CheckCertJob{}
var checkSecretJob = &jobs.CheckSecretJob{}
var env string

func loadEnv() {
	// outside of local environment, variables should be
	// OS environment variables
	env = os.Getenv("ENV")
	if err := godotenv.Load(); err != nil && env == "" {
		log.Fatal(fmt.Printf("Error loading .env file: %s", err))
	}
	env = os.Getenv("ENV") // reload env from .env file
}

func getJobNotifier() jobs.Notifier {
	result := &jobs.WebHookNotifier{}

	webhookType, _ := os.LookupEnv("WEBHOOK_TYPE")
	WebhookUrl, _ := os.LookupEnv("WEBHOOK_URL")
	messageUrl, _ := os.LookupEnv("MESSAGE_URL")
	messageTitle, _ := os.LookupEnv("MESSAGE_TITLE")
	messageBody, _ := os.LookupEnv("MESSAGE_BODY")
	messageMentions, _ := os.LookupEnv("MESSAGE_MENTIONS")

	result.Init(jobs.Notifiers[webhookType], WebhookUrl, messageTitle, messageBody, messageUrl, messageMentions)

	return result
}

func startJobs(siteList []models.CheckCertItem) {
	schedule, ok := os.LookupEnv("CHECK_CERT_JOB_SCHEDULE")

	if ok {
		log.Printf("Starting job engine with cron: %s", schedule)
		level, _ := os.LookupEnv("CHECK_CRED_JOB_NOTIFICATION_LEVEL")
		warningDays := getCertExpirationWarningDays()
		err := checkCertJob.Init(schedule, level, warningDays, siteList, getJobNotifier())
		if err == nil {
			checkCertJob.Start()
			log.Print("Job engine started")
		} else {
			log.Printf("Error starting job: %s", err)
		}
	} else {
		log.Println("No schedule defined for jobs")
	}
}

func getCertExpirationWarningDays() int {
	warningDaysConfig, _ := os.LookupEnv("CERT_WARNING_VALIDITY_DAYS")
	warningDays, _ := strconv.Atoi(warningDaysConfig)

	if warningDays > 0 {
		return warningDays
	}

	return 30
}

func getSecretWarningValidityDays() int {
	warningDaysConfig, _ := os.LookupEnv("SECRET_WARNING_VALIDITY_DAYS")
	warningDays, _ := strconv.Atoi(warningDaysConfig)

	if warningDays > 0 {
		return warningDays
	}

	return 30
}

func getCORSOrigins() string {
	corsOrigins, ok := os.LookupEnv("CORS_ORIGINS")
	if ok {
		return corsOrigins
	}

	return ""
}

func startSecretJobs(secretList []models.CheckSecretItem) {
	schedule, ok := os.LookupEnv("CHECK_SECRET_JOB_SCHEDULE")

	if ok {
		log.Printf("Starting secret job engine with cron: %s", schedule)
		level, _ := os.LookupEnv("CHECK_CRED_JOB_NOTIFICATION_LEVEL")
		warningDays := getSecretWarningValidityDays()
		err := checkSecretJob.Init(schedule, level, warningDays, secretList, getJobNotifier())
		if err == nil {
			checkSecretJob.Start()
			log.Print("Secret job engine started")
		} else {
			log.Printf("Error starting secret job: %s", err)
		}
	} else {
		log.Println("No schedule defined for secret jobs")
	}
}

func stopJobs() {
	checkCertJob.Stop()
	checkSecretJob.Stop()
}

func startWebServer(siteList []models.CheckCertItem, secretList []models.CheckSecretItem) {
	headless, _ := os.LookupEnv("HEADLESS")

	if headless == "true" {
		log.Println("Running in headless mode. Skipping web server start.")
		return
	}

	handlers := &handlers.Handlers{}
	handlers.CertList = siteList
	handlers.ExpirationWarningDays = getCertExpirationWarningDays()
	handlers.SecretList = secretList
	handlers.SecretWarningValidityDays = getSecretWarningValidityDays()
	handlers.CORSOrigins = getCORSOrigins()

	router := http.NewServeMux()

	router.HandleFunc("GET /api/check-cert", handlers.CheckCertStatus)
	router.HandleFunc("GET /api/cert-list", handlers.GetCertList)
	router.HandleFunc("GET /api/secret-list", handlers.GetSecretList)
	router.HandleFunc("GET /api/check-secret", handlers.CheckSecretStatus)
	router.HandleFunc("GET /health", handlers.HealthCheck)

	if handlers.CORSOrigins != "" {
		router.HandleFunc("OPTIONS /api/", handlers.CORS)
	}

	router.HandleFunc("GET /", handlers.Index)
	router.HandleFunc("GET /item", handlers.GetItem)
	router.HandleFunc("GET /itemDetail", handlers.GetItemDetail)
	router.HandleFunc("GET /secrets-panel", handlers.GetSecretsPanel)
	router.HandleFunc("GET /secret-item", handlers.GetSecretItem)
	router.HandleFunc("GET /secret-item-detail", handlers.GetSecretItemDetail)
	router.HandleFunc("GET /empty", handlers.GetEmpty)
	router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./public/"))))

	logRouter := midlewares.NewLogger(router)

	hostPort, ok := os.LookupEnv("WEB_HOST_PORT")
	if !ok {
		hostPort = ":8000"
	}

	certFile, useTls := os.LookupEnv("TLS_CERT_FILE")

	certKeyFile, ok := os.LookupEnv("TLS_CERT_KEY_FILE")
	useTls = useTls && ok

	log.Printf("Starting TLS server on port: %s; use tls: %t", hostPort, useTls)

	srv := &http.Server{
		Addr: hostPort,
	}

	srv.Handler = logRouter

	go func() {
		var err error = nil
		if useTls {
			err = srv.ListenAndServeTLS(certFile, certKeyFile)
		} else {
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Web Server Started")
}

func runOnce(siteList []models.CheckCertItem, secretList []models.CheckSecretItem, done chan os.Signal) {
	certSchedule, _ := os.LookupEnv("CHECK_CERT_JOB_SCHEDULE")
	secretSchedule, _ := os.LookupEnv("CHECK_SECRET_JOB_SCHEDULE")
	headless, _ := os.LookupEnv("HEADLESS")

	if certSchedule != "" || secretSchedule != "" || headless != "true" {
		return
	}

	log.Print("Running the checkCertJob once")
	level, _ := os.LookupEnv("CHECK_CRED_JOB_NOTIFICATION_LEVEL")
	warningDays := getCertExpirationWarningDays()
	err := checkCertJob.Init("* * * * *", level, warningDays, siteList, getJobNotifier())
	if err == nil {
		checkCertJob.RunNow()
	} else {
		log.Fatalf("Error running the checkCertJob once: %s", err)
	}

	err = checkSecretJob.Init("* * * * *", level, warningDays, secretList, getJobNotifier())
	if err == nil {
		checkSecretJob.RunNow()
	} else {
		log.Fatalf("Error running the checkSecretJob once: %s", err)
	}

	// We don't want to wait on anything so do a graceful exist
	done <- syscall.SIGQUIT
}

func main() {
	loadEnv()

	siteList := services.GetConfigCerts()
	secretList := services.GetConfigSecrets()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	startWebServer(siteList, secretList)
	startJobs(siteList)
	startSecretJobs(secretList)
	runOnce(siteList, secretList, done)

	<-done
	log.Print("Stopping jobs...")
	stopJobs()

	log.Print("All done. Bye!")
}
