// handlers/terminal.go
package handlers

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// TerminalMessage - WebSocket 메시지 구조
type TerminalMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ResizeMessage - 터미널 크기 조정 메시지
type ResizeMessage struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// LocalTerminal - 실제 터미널 세션 관리
type LocalTerminal struct {
	cmd       *exec.Cmd       // 실행 중인 명령어
	pty       *os.File        // 가상 터미널
	conn      *websocket.Conn // WebSocket 연결
	done      chan bool       // 종료 신호
	sessionID string          // 세션 ID
}

// NewLocalTerminal - 새 로컬 터미널 생성
func NewLocalTerminal(conn *websocket.Conn, sessionID string) (*LocalTerminal, error) {
	log.Printf("터미널 생성 시작: %s", sessionID) //디버깅 확인

	// OS에 따른 셸 명령어 결정
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		log.Printf("Windows 환경에서 cmd 실행") //디버깅 확인
		cmd = exec.Command("cmd")
	} else {
		// Linux/macOS
		shellPath := "/bin/bash"
		if _, err := os.Stat(shellPath); err != nil {
			shellPath = "/bin/sh"
		}
		log.Printf("Unix 환경에서 %s 실행", shellPath) //디버깅 확인
		cmd = exec.Command(shellPath)
	}

	// 환경 변수 설정
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"PS1=🐳 container:$ ",
		"LANG=en_US.UTF-8",
	)
	log.Printf("환경 변수 설정 완료") //디버깅 확인

	// PTY 생성 및 명령어 시작
	log.Printf("PTY 생성 시도 중..") //디버깅 확인
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		log.Printf("PTY 생성 실패: %v", err) //디버깅 확인
		return nil, fmt.Errorf("PTY 시작 실패: %v", err)
	}
	log.Printf("PTY 생성 성공 (PID: %d)", cmd.Process.Pid) //디버깅 확인

	terminal := &LocalTerminal{
		cmd:       cmd,
		pty:       ptyFile,
		conn:      conn,
		done:      make(chan bool),
		sessionID: sessionID,
	}

	log.Printf("🖥️ 새 터미널 세션 시작: %s (PID: %d)", sessionID, cmd.Process.Pid)

	// 백그라운드 고루틴 시작
	log.Printf("백그라운드 go루틴 시작") //디버깅 확인
	go terminal.handlePtyOutput()
	go terminal.handleWebSocketInput()
	go terminal.monitorProcess()

	log.Printf("터미널 생성 및 초기화 완료: %s", sessionID) //디버깅 확인
	return terminal, nil
}

// handlePtyOutput - PTY 출력을 WebSocket으로 전송
func (lt *LocalTerminal) handlePtyOutput() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-lt.done:
			return
		default:
			// PTY에서 데이터 읽기
			n, err := lt.pty.Read(buffer)
			if err != nil {
				if err == io.EOF {
					log.Printf("터미널 출력 종료: %s", lt.sessionID)
				} else {
					log.Printf("PTY 읽기 오류: %v", err)
				}
				lt.Close()
				return
			}

			// WebSocket으로 출력 데이터 전송
			if err := lt.conn.WriteJSON(TerminalMessage{
				Type: "output",
				Data: string(buffer[:n]),
			}); err != nil {
				log.Printf("WebSocket 출력 전송 실패: %v", err)
				lt.Close()
				return
			}
		}
	}
}

// handleWebSocketInput - WebSocket 입력을 PTY로 전송
func (lt *LocalTerminal) handleWebSocketInput() {
	for {
		select {
		case <-lt.done:
			return
		default:
			var message TerminalMessage
			err := lt.conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket 입력 오류: %v", err)
				} else {
					log.Printf("클라이언트 연결 종료: %s", lt.sessionID)
				}
				lt.Close()
				return
			}

			// 메시지 타입에 따른 처리
			switch message.Type {
			case "input":
				// 사용자 입력을 PTY로 전송
				if input, ok := message.Data.(string); ok {
					_, err := lt.pty.WriteString(input)
					if err != nil {
						log.Printf("PTY 입력 전송 실패: %v", err)
						lt.Close()
						return
					}
				}

			case "resize":
				// 터미널 크기 조정
				if resizeData, ok := message.Data.(map[string]interface{}); ok {
					if cols, hasC := resizeData["cols"].(float64); hasC {
						if rows, hasR := resizeData["rows"].(float64); hasR {
							lt.resize(int(cols), int(rows))
						}
					}
				}

			case "ping":
				// Ping-Pong
				pongMessage := TerminalMessage{
					Type: "pong",
					Data: "터미널 연결 정상",
				}
				lt.conn.WriteJSON(pongMessage)

			case "command":
				// 특별 명령어 처리
				if cmdStr, ok := message.Data.(string); ok {
					lt.handleSpecialCommand(cmdStr)
				}

			default:
				log.Printf("알 수 없는 메시지 타입: %s", message.Type)
			}
		}
	}
}

// handleSpecialCommand - 특별 명령어 처리
func (lt *LocalTerminal) handleSpecialCommand(command string) {
	switch command {
	case "clear":
		// 화면 지우기
		lt.pty.WriteString("clear\n")
	case "help":
		// 도움말
		helpText := `터미널 도움말`
		lt.pty.WriteString(fmt.Sprintf("echo '%s'\n", helpText))
	default:
		// 일반 명령어는 그대로 전송
		lt.pty.WriteString(command + "\n")
	}
}

// monitorProcess - 프로세스 상태 모니터링
func (lt *LocalTerminal) monitorProcess() {
	// 프로세스 종료 대기
	err := lt.cmd.Wait()

	exitCode := 0
	if lt.cmd.ProcessState != nil {
		exitCode = lt.cmd.ProcessState.ExitCode()
	}

	log.Printf("🏁 프로세스 종료: %s (PID: %d, 코드: %d, 에러: %v)",
		lt.sessionID, lt.cmd.Process.Pid, exitCode, err)

	// 종료 메시지 전송
	exitMessage := TerminalMessage{
		Type: "exit",
		Data: map[string]interface{}{
			"message":   "터미널 세션이 종료되었습니다",
			"code":      exitCode,
			"sessionId": lt.sessionID,
		},
	}

	lt.conn.WriteJSON(exitMessage)
	lt.Close()
}

// resize - 터미널 크기 조정
func (lt *LocalTerminal) resize(cols, rows int) {
	log.Printf("🔧 터미널 크기 조정: %dx%d (세션: %s)", cols, rows, lt.sessionID)

	if runtime.GOOS != "windows" {
		// Unix 계열에서만 동작
		type winsize struct {
			rows, cols, xpixel, ypixel uint16
		}

		ws := winsize{
			rows:   uint16(rows),
			cols:   uint16(cols),
			xpixel: 0,
			ypixel: 0,
		}

		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			lt.pty.Fd(),
			syscall.TIOCSWINSZ,
			uintptr(unsafe.Pointer(&ws)),
		)

		if errno != 0 {
			log.Printf("터미널 크기 조정 실패: %v", errno)
		}
	} else {
		// Windows에서는 크기 조정이 복잡함
		log.Printf("Windows에서는 터미널 크기 조정이 지원되지 않습니다")
	}
}

// Close - 터미널 세션 종료
func (lt *LocalTerminal) Close() {
	select {
	case <-lt.done:
		// 이미 닫힘
		return
	default:
		log.Printf("터미널 세션 종료 중: %s", lt.sessionID)
		close(lt.done)

		// 리소스 정리
		if lt.pty != nil {
			lt.pty.Close()
		}

		if lt.cmd != nil && lt.cmd.Process != nil {
			// 프로세스 강제 종료
			if runtime.GOOS == "windows" {
				lt.cmd.Process.Kill()
			} else {
				// Unix 계열에서는 SIGTERM 먼저 시도
				lt.cmd.Process.Signal(syscall.SIGTERM)
				// 잠시 대기 후 SIGKILL
				go func() {
					// time.Sleep(2 * time.Second)
					if lt.cmd.Process != nil {
						lt.cmd.Process.Kill()
					}
				}()
			}
		}

		if lt.conn != nil {
			// 종료 메시지 전송 시도
			finalMessage := TerminalMessage{
				Type: "system",
				Data: "터미널 세션이 종료되었습니다",
			}
			lt.conn.WriteJSON(finalMessage)
			lt.conn.Close()
		}

		log.Printf("터미널 세션 정리 완료: %s", lt.sessionID)
	}
}

// IsAlive - 터미널이 살아있는지 확인
func (lt *LocalTerminal) IsAlive() bool {
	select {
	case <-lt.done:
		return false
	default:
		return lt.cmd != nil && lt.cmd.Process != nil
	}
}

// SendMessage - 터미널에 메시지 전송 (외부에서 호출 가능)
func (lt *LocalTerminal) SendMessage(msgType string, data interface{}) error {
	message := TerminalMessage{
		Type: msgType,
		Data: data,
	}
	return lt.conn.WriteJSON(message)
}

// WriteToTerminal - 터미널에 직접 텍스트 작성
func (lt *LocalTerminal) WriteToTerminal(text string) error {
	if lt.pty == nil {
		return fmt.Errorf("터미널이 초기화되지 않았습니다")
	}
	_, err := lt.pty.WriteString(text)
	return err
}

// GetInfo - 터미널 정보 반환
func (lt *LocalTerminal) GetInfo() map[string]interface{} {
	info := map[string]interface{}{
		"sessionId": lt.sessionID,
		"alive":     lt.IsAlive(),
	}

	if lt.cmd != nil && lt.cmd.Process != nil {
		info["pid"] = lt.cmd.Process.Pid
	}

	return info
}
