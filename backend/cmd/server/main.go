package main

import (
	"log"
	"net/http"
	"os"

	"authgate/internal/api"
	"authgate/internal/auth"
	"authgate/internal/store"
)

func main() {
	cfg := api.Config{
		FeaturesDir:  "features",
		ReportPath:   "../bdd-report.json",
		EnableRunner: os.Getenv("ENABLE_BDD_RUNNER") == "1",
	}
	log.Println("AuthGate listening on :8080 (bdd runner enabled:", cfg.EnableRunner, ")")
	log.Fatal(http.ListenAndServe(":8080", api.NewRouter(store.New(), auth.New(), cfg)))
}
