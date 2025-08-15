package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/Heo-YJ/teleport-opensource/handlers"
)

type Container struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Status string            `json:"status"`
	Labels map[string]string `json:"labels"`
}

func main() {
	// 라우터 생성
	r := mux.NewRouter()

	//핸들러 인스턴스 생성
	teleportHandler := handlers.NewTeleportHandler()

	//API 라우트 설정
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", healthCheck).Methods("GET")
	api.HandleFunc("/containers", getContainers).Methods("GET")
	api.HandleFunc("/containers/{containerId}", teleportHandler.HandleGetContainer).Methods("GET")
	api.HandleFunc("/containers/{containerId}/connect", connectContainer).Methods("POST")
	api.HandleFunc("/terminal/sessions", teleportHandler.HandleGetTerminalSessions).Methods("GET")
	api.HandleFunc("/ws/terminal/{containerId}", teleportHandler.HandleTerminalWebSocket).Methods("GET")

	//CORS 설정 (프론트엔드와 연동용)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"}, //React 개발 서버
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	fmt.Println("Container SSH System Backend")
	fmt.Println("API Server: http://localhost:8080")
	fmt.Println("Health Check: http://localhost:8080/api/health")
	fmt.Println("Containers:http://localhost:8080/api/containers")

	log.Println("서버가 :8080 포트에서 시작됩니다...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":   "ok",
		"message":  "Container SSH System is running",
		"teleport": "connected",
	}
	json.NewEncoder(w).Encode(response)
}

func getContainers(w http.ResponseWriter, r *http.Request) {
	//임시 Mock 데이터 (나중에 Teleport API로 교체)
	containers := []Container{
		{
			ID:     "web-001",
			Name:   "nginx-server",
			Status: "running",
			Labels: map[string]string{
				"team": "frontend",
				"env":  "production",
			},
		},
		{
			ID:     "db-001",
			Name:   "postgres-db",
			Status: "running",
			Labels: map[string]string{
				"team": "backend",
				"env":  "production",
			},
		},
		{
			ID:     "api-001",
			Name:   "backend-api",
			Status: "stopped",
			Labels: map[string]string{
				"team": "backend",
				"env":  "development",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"containers": containers,
		"total":      len(containers),
	})
}

func connectContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["containerId"]

	// 임시 응답 (나중에 실제 SSH 연결로 교체)
	response := map[string]string{
		"status":       "connecting",
		"container_id": containerID,
		"message":      "SSH connection initiated",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
