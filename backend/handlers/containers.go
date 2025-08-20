package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/gravitational/teleport/api/client"
)

// WebSocket 업그레이더 설정
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		//프론트엔드에서 오는 요청 허용
		return true
	},
}

type TeleportHandler struct {
	client          *client.Client
	terminalHandler *TerminalHandler
}

// ---- [임시 스텁: TerminalHandler / Session] ----
// 8/21 수정
type TerminalHandler struct {
	sessions  map[string]*Session       // 세션 저장소
	terminals map[string]*LocalTerminal // 터미널 저장소
}

// 프론트엔드와 일치하게!
type Session struct {
	ID          string          `json:"id"`
	ContainerID string          `json:"containerId"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
	Connection  *websocket.Conn `json:"-"` // JSON에서 제외
}

func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{
		sessions:  make(map[string]*Session), // 세션 맵 초기화
		terminals: make(map[string]*LocalTerminal),
	}
}

func (t *TerminalHandler) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	// HTTP 응답으로 상태 알림 -> HTTP를 WebSocket으로 업그레이드
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		return
	}

	log.Println("WebSocket 연결 성공")

	// 세션ID 생성
	vars := mux.Vars(r)
	containerID := vars["containerId"]
	if containerID == "" {
		containerID = "default"
	}

	sessionID := fmt.Sprintf("terminal-%s-%d", containerID, time.Now().Unix())
	log.Printf("세션 ID 생성: %s", sessionID)

	// 연결 성공 메시지 전송
	welcomMsg := TerminalMessage{
		Type: "system",
		Data: map[string]interface{}{
			"message":   "터미널 연결이 성공했습니다. 메시지를 입력해주세요.",
			"sessionId": sessionID,
			"time":      time.Now().Format("15:04:05"),
		},
	}

	if err := conn.WriteJSON(welcomMsg); err != nil {
		log.Printf("환영 메시지 전송 실패: %v", err)
		conn.Close()
		return
	}

	log.Printf("터미널 생성 시도 중..") //디버깅 확인

	// 메시지 루프
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 연결 오류: %v", err)
			} else {
				log.Println("클라이언트가 연결을 종료했습니다.")
			}
			break
		}
		log.Printf("받은 메시지: %+v", message)

		// 메시지 타입에 따른 처리
		if msgType, exists := message["type"]; exists {
			switch msgType {
			case "command":
				//명령어 처리
				if cmd, ok := message["data"].(string); ok {
					t.handleCommand(conn, cmd)
				}
			case "ping":
				// ping-pong 처리
				pongMsg := map[string]string{
					"type":    "pong",
					"message": "서버 응답 정상",
					"time":    time.Now().Format("15:04:05"),
				}
				conn.WriteJSON(pongMsg)
			default:
				// Echo 처리
				t.handleEcho(conn, message)
			}
		}
		log.Println("WebSocket 연결 종료")
	}

	// 로컬 터미널 생성
	terminal, err := NewLocalTerminal(conn, sessionID)
	if err != nil {
		log.Printf("터미널 생성 실패: %v", err)

		//에러 메시지 전송
		errorMsg := TerminalMessage{
			Type: "error",
			Data: map[string]interface{}{
				"message": fmt.Sprintf("터미널 생성 실패: %v", err),
				"time":    time.Now().Format("15:04:05"),
			},
		}
		conn.WriteJSON(errorMsg)
		conn.Close()
		return
	}

	log.Printf("터미널 생성 성공: %s", sessionID) //디버깅 확인

	// 터미널을 맵에 저장
	t.terminals[sessionID] = terminal
	log.Printf("터미널 맵에 저장 완료") //디버깅 확인

	// 터미널 종료 대기
	select {
	case <-terminal.done:
		log.Printf("터미널 세션 완료: %s", sessionID)
	}

	delete(t.terminals, sessionID)
	log.Printf("터미널 세션 정리 완료: %s", sessionID)
}

// 명령어 처리 함수
func (t *TerminalHandler) handleCommand(conn *websocket.Conn, command string) {
	log.Printf("명령어 실행: %s", command)

	var response map[string]string

	switch command {
	case "help":
		response = map[string]string{
			"type":    "output",
			"message": "사용 가능한 명령어: \n- help: 도움말\n- date: 현재 시간\n- echo [텍스트]: 에코\n- clear: 화면 지우기",
			"time":    time.Now().Format("15:04:05"),
		}
	case "date":
		response = map[string]string{
			"type":    "output",
			"message": fmt.Sprintf("현재 시간: %s", time.Now().Format("2025-08-21 15:04:05")),
			"time":    time.Now().Format("15:04:05"),
		}
	case "clear":
		response = map[string]string{
			"type":    "clear",
			"message": "화면을 지웠습니다.",
			"time":    time.Now().Format("15:04:05"),
		}
	default:
		if len(command) > 5 && command[:4] == "echo" {
			echoText := command[5:]
			response = map[string]string{
				"type":    "output",
				"message": echoText,
				"time":    time.Now().Format("15:04:05"),
			}
		} else {
			response = map[string]string{
				"type":    "error",
				"message": fmt.Sprintf("알 수 없는 명령어: %s (help를 입력해주세요)", command),
				"time":    time.Now().Format("15:04:05"),
			}
		}
	}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("명령어 응답 전송 실패: %v", err)
	}
}

// Echo 처리 함수
func (t *TerminalHandler) handleEcho(conn *websocket.Conn, message map[string]interface{}) {
	response := map[string]interface{}{
		"type":     "echo",
		"original": message,
		"message":  fmt.Sprintf("Echo: %+v", message),
		"time":     time.Now().Format("15:04:05"),
	}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Echo 응답 전송 실패: %v", err)
	}
}

// 활성 터미널 관리
func (t *TerminalHandler) GetActiveTerminals() map[string]*LocalTerminal {
	activeTerminals := make(map[string]*LocalTerminal)

	for sessionID, terminal := range t.terminals {
		if terminal.IsAlive() {
			activeTerminals[sessionID] = terminal
		} else {
			delete(t.terminals, sessionID)
		}
	}
	return activeTerminals
}

// 특정 터미널 종료
func (t *TerminalHandler) CloseTerminal(sessionID string) bool {
	if terminal, exists := t.terminals[sessionID]; exists {
		terminal.Close()
		delete(t.terminals, sessionID)
		log.Printf("터미널 강제 종료: %s", sessionID)
		return true
	}
	return false
}

// 전체 터미널 종료
func (t *TerminalHandler) CloseAllTerminals() {
	log.Println("모든 터미널 세션 종료 중..")

	for sessionID, terminal := range t.terminals {
		terminal.Close()
		delete(t.terminals, sessionID)
	}
	log.Println("모든 터미널 세션 종료 완료")
}

// 세션 관리
func (t *TerminalHandler) GetActiveSessions() []Session {
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

// Mock 컨테이너 데이터 (3단계에서 실제 Teleport API로 교체 예정)
func (h *TeleportHandler) getMockContainers() []ContainerInfo {
	return []ContainerInfo{
		{
			ID:     "web-001",
			Name:   "nginx-server",
			Status: "online",
			Labels: map[string]string{
				"environment": "production",
				"service":     "web",
				"team":        "frontend",
			},
			NodeAddr: "0.0.0.0:3022",
		},
		{
			ID:     "db-001",
			Name:   "postgres-db",
			Status: "online",
			Labels: map[string]string{
				"environment": "production",
				"service":     "database",
				"team":        "backend",
			},
			NodeAddr: "0.0.0.0:3022",
		},
		{
			ID:     "api-001",
			Name:   "backend-api",
			Status: "offline",
			Labels: map[string]string{
				"environment": "development",
				"service":     "api",
				"team":        "backend",
			},
			NodeAddr: "0.0.0.0:3022",
		},
	}
}

// 생성자 함수
func NewTeleportHandler() *TeleportHandler {
	// Teleport 클라이언트 설정
	/* 나중에 실제 Teleport 클라이언트 추가 예정
	config := config.LoadTeleportConfig()

	clientConfig := client.Config{
		Addrs: []string{config.AuthServer},
	}

	teleportClient, err := client.New(context.Background(), clientConfig)
	if err != nil {
		log.Printf("Teleport 클라이언트 생성 실패: %v", err)
		teleportClient = nil //개발 중에는 nil로 두고 Mock데이터 사용
	}
	*/
	return &TeleportHandler{
		// client:          teleportClient,
		terminalHandler: NewTerminalHandler(),
	}
}

// Teleport API를 통한 실제 컨테이너 목록 조회 (구현 예정)
func (h *TeleportHandler) GetTeleportContainers(ctx context.Context) ([]ContainerInfo, error) {
	//현재는 Mock 데이터 반환
	/* 나중에 다시 구현 예정. 일단 Mock 데이터로!
	if h.client == nil {
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

		return h.getMockContainers(), nil
	}
	*/
	return h.getMockContainers(), nil
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
