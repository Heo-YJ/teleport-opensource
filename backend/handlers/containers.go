package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type TeleportHandler struct {
	// (나중에 Teleport 패키지 설치 후 주석 해제)
	// client teleport.Client
	terminalHandler *TerminalHandler
}

// ---- [임시 스텁: TerminalHandler / Session] ----
type TerminalHandler struct {
	sessions map[string]*Session // 세션 저장소
}

// 프론트엔드와 일치하게!
type Session struct {
	ID          string    `json:"id"`
	ContainerID string    `json:"containerId"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{
		sessions: make(map[string]*Session), // 세션 맵 초기화
	}
}

// 기본 응답만 추가!
func (t *TerminalHandler) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// TODO: 실제 WebSocket 업그레이드/브릿지 로직 (업그레이드 구현 예정)
	// 현재는 HTTP 응답으로 상태 알림
	log.Println("WebSocket 연결 시도됨")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)

	response := map[string]string{
		"message": "WebSocket 터미널 기능이 아직 구현되지 않았습니다.",
		"status":  "not_implemented",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("응답 인코딩 실패: %v", err)
	}
}

// 세션 관리
func (t *TerminalHandler) GetActiveSessions() []Session {
	// TODO: 실제 세션 목록 반환
	sessions := make([]Session, 0, len(t.sessions))
	for _, session := range t.sessions {
		sessions = append(sessions, *session)
	}

	// 개발용 Mock 세션 데이터
	if len(sessions) == 0 {
		mockSession := Session{
			ID:          "mock-session-1",
			ContainerID: "teleport-node-1",
			Status:      "connected",
			CreatedAt:   time.Now().Add(-30 * time.Minute),
		}
		sessions = append(sessions, mockSession)
	}

	return sessions
}

// 세션 추가 메서드
func (t *TerminalHandler) AddSession(containerID string) *Session {
	sessionID := "session-" + containerID + "-" + time.Now().Format("20060102150405")
	session := &Session{
		ID:          sessionID,
		ContainerID: containerID,
		Status:      "connecting",
		CreatedAt:   time.Now(),
	}

	t.sessions[sessionID] = session
	log.Printf("새 세션 생성: %s (컨테이너: %s)", sessionID, containerID)
	return session
}

// 세션 제거 메서드
func (t *TerminalHandler) RemoveSession(sessionID string) {
	if _, exists := t.sessions[sessionID]; exists {
		delete(t.sessions, sessionID)
		log.Printf("세션 제거: %s", sessionID)
	}
}

// ------------------------------------------------

type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Labels   map[string]string `json:"labels"`
	NodeAddr string            `json:"node_addr"`
}

type ContainerListResponse struct {
	Containers []ContainerInfo `json:"containers"`
	Total      int             `json:"total"`
}

// 생성자 함수
func NewTeleportHandler() *TeleportHandler {
	// TOTO: 실제 Teleport 클라이언트 초기화
	/*
		config := client.Config{
			Addrs: []string{"teleport.example.com:443"},
			Credentials: []client.Credentials{...},
		}
		teleportClient, err := client.New(ctx, config)
	*/
	return &TeleportHandler{
		// client: nil, (현재는 주석 처리)
		terminalHandler: NewTerminalHandler(),
	}
}

// Teleport API를 통한 실제 컨테이너 목록 조회 (구현 예정)
func (h *TeleportHandler) GetTeleportContainers(ctx context.Context) ([]ContainerInfo, error) {
	//TODO: Teleport API 통합
	//현재는 Mock 데이터 반환

	containers := []ContainerInfo{
		{
			ID:     "teleport-node-1",
			Name:   "production-web",
			Status: "online",
			Labels: map[string]string{
				"environment": "production",
				"service":     "web",
			},
			NodeAddr: "0.0.0.0:3022",
		},
	}

	return containers, nil
}

// HTTP 핸들러: 컨테이너 목록 조회
func (h *TeleportHandler) HandleGetContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.GetTeleportContainers(r.Context())
	if err != nil {
		log.Printf("컨테이너 목록 조회 실패: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := ContainerListResponse{
		Containers: containers,
		Total:      len(containers),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON 인코딩 실패: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("컨테이너 목록 반환: %d개", len(containers))
}

// WebSocket을 통한 터미널 연결
func (h *TeleportHandler) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["containerId"]

	if containerID == "" {
		log.Println("컨테이너 ID가 제공되지 않음")
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	// 컨테이너 존재 여부 확인
	containers, err := h.GetTeleportContainers(r.Context())
	if err != nil {
		log.Printf("컨테이너 조회 실패: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var targetContainer *ContainerInfo
	for i := range containers {
		if containers[i].ID == containerID {
			targetContainer = &containers[i]
			break
		}
	}

	if targetContainer == nil {
		log.Printf("컨테이너를 찾을 수 없음: %s", containerID)
		http.Error(w, "Container not found", http.StatusNotFound)
		return
	}

	if targetContainer.Status != "online" {
		log.Printf("컨테이너가 온라인이 아님: %s (상태: %s)", containerID, targetContainer.Status)
		http.Error(w, "Container is not online", http.StatusBadRequest)
		return
	}

	log.Printf("터미널 WebSocket 연결 요청: 컨테이너 %s (%s)", targetContainer.Name, containerID)

	// 세션 생성 후 터미널 핸들러에게 위임
	session := h.terminalHandler.AddSession(containerID)
	log.Printf("세션 생성됨: %s", session.ID)

	h.terminalHandler.HandleWebSocketConnection(w, r)
}

// 특정 컨테이너 상세 정보 조회 (단건)
func (h *TeleportHandler) HandleGetContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["containerId"]

	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	containers, err := h.GetTeleportContainers(r.Context())
	if err != nil {
		log.Printf("컨테이너 조회 실패: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for i := range containers {
		if containers[i].ID == containerID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(containers[i])
			return
		}
	}

	http.Error(w, "Container not found", http.StatusNotFound)
}

// 활성 터미널 세션 목록 조회
func (h *TeleportHandler) HandleGetTerminalSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.terminalHandler.GetActiveSessions()

	sessionList := make([]map[string]interface{}, 0, len(sessions))
	for _, session := range sessions {
		sessionInfo := map[string]interface{}{
			"id":          session.ID,
			"containerId": session.ContainerID,
			"status":      "connected",
			"createdAt":   "2025-01-01T00:00:00Z", //하드코딩으로 임시 대체
		}
		sessionList = append(sessionList, sessionInfo)
	}

	response := map[string]interface{}{
		"sessions": sessionList,
		"total":    len(sessionList),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON 인코딩 실패: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("활성 터미널 세션 반환: %d개", len(sessionList))
}

/*
//TODO: WebSocket 업그레이드 및 Teleport SSH 세션 연결
log.Println("Terminal WebSocket connection requested")

w.WriteHeader(http.StatusNotImplemented)
json.NewEncoder(w).Encode(map[string]string{
	"message": "Terminal WebSocket not yet implemented",
})
*/
