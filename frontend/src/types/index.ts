export interface Container {
    id: string;
    name: string;
    status: ContainerStatus;
    labels: Record<string, string>;
    nodeAddr?: string;
    lastSeen?: string;
    uptime?: string;
    // 추가: 웹 터미널 관련 정보
    image?: string;
    created?: string;
    ports?: string[];
  }
  
  export type ContainerStatus = 'running' | 'online' |'stopped' | 'pending' | 'error' | 'unknown';
  
  export interface ContainerConnection {
    containerId: string;
    sessionId: string;
    status: ConnectionStatus;
    startTime: string;
    endTime?: string;
  }
  
  export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';
  
  export interface SSHSession {
    id: string;
    containerId: string;
    userId: string;
    startTime: string;
    endTime?: string;
    commands: string[];
    recordingUrl?: string;
  }
  
  export interface User {
    id: string;
    username: string;
    email: string;
    roles: string[];
    permissions: Permission[];
  }
  
  export interface Permission {
    resource: string;
    actions: string[];
  }
  
  export interface AccessRequest {
    id: string;
    userId: string;
    containerId: string;
    reason: string;
    status: AccessRequestStatus;
    requestTime: string;
    approvedBy?: string;
    approvedTime?: string;
    expiryTime?: string;
  }
  
  export type AccessRequestStatus = 'pending' | 'approved' | 'denied' | 'expired';
  
  export interface AuditLog {
    id: string;
    timestamp: string;
    userId: string;
    action: string;
    resource: string;
    details: Record<string, any>;
    ipAddress: string;
    userAgent: string;
  }
  
  export interface ApiResponse<T> {
    data: T;
    success: boolean;
    message?: string;
    error?: string;
  }
  
  // 수정: WebSocket 메시지 타입 확장
  export interface WebSocketMessage {
    type: 'input' | 'output' | 'resize' | 'close' | 'error' | 'data';
    data?: string | ResizeData;
    payload?: any;
  }
  
  // 새로 추가: 터미널 관련 타입들
  export interface TerminalSession {
    id: string;
    containerId: string;
    containerName: string;
    status: 'connecting' | 'connected' | 'disconnected' | 'error';
    createdAt: string;
    userId?: string;
    lastActivity?: string;
  }
  
  export interface ResizeData {
    cols: number;
    rows: number;
  }
  
  export interface TerminalConfig {
    fontSize: number;
    fontFamily: string;
    theme: {
      background: string;
      foreground: string;
      cursor: string;
      selectionBackground: string;
      black?: string;
      red?: string;
      green?: string;
      yellow?: string;
      blue?: string;
      magenta?: string;
      cyan?: string;
      white?: string;
      brightBlack?: string;
      brightRed?: string;
      brightGreen?: string;
      brightYellow?: string;
      brightBlue?: string;
      brightMagenta?: string;
      brightCyan?: string;
      brightWhite?: string;
    };
    cursorBlink?: boolean;
    scrollback?: number;
  }
  
  // 새로 추가: 컨테이너 목록 응답 타입
  export interface ContainerListResponse {
    containers: Container[];
    total: number;
    page?: number;
    pageSize?: number;
  }
  
  // 새로 추가: 터미널 세션 목록 응답 타입
  export interface TerminalSessionListResponse {
    sessions: TerminalSession[];
    total: number;
  }
  
  // 새로 추가: 터미널 연결 요청 타입
  export interface TerminalConnectionRequest {
    containerId: string;
    containerName: string;
    userId?: string;
    cols?: number;
    rows?: number;
  }
  
  // 새로 추가: 터미널 이벤트 타입
  export interface TerminalEvent {
    type: 'session_start' | 'session_end' | 'command_executed' | 'file_accessed' | 'error';
    sessionId: string;
    containerId: string;
    userId: string;
    timestamp: string;
    details: Record<string, any>;
  }
  
  // Teleport specific types
  export interface TeleportNode {
    id: string;
    name: string;
    addr: string;
    labels: Record<string, string>;
    connected: boolean;
    lastHeartbeat: string;
  }
  
  export interface TeleportRole {
    name: string;
    allow: {
      logins: string[];
      node_labels: Record<string, string[]>;
    };
    deny?: {
      logins: string[];
      node_labels: Record<string, string[]>;
    };
  }
  
  export interface TeleportUser {
    name: string;
    roles: string[];
    traits: Record<string, string[]>;
  }
  
  // 새로 추가: API 엔드포인트 관련 타입들
  export interface ApiEndpoints {
    containers: '/api/containers';
    terminalSessions: '/api/terminal/sessions';
    websocketTerminal: '/ws/terminal';
  }
  
  // 새로 추가: 에러 타입
  export interface AppError {
    code: string;
    message: string;
    details?: Record<string, any>;
    timestamp: string;
  }
  
  // 새로 추가: 터미널 상태 관리를 위한 타입
  export interface TerminalState {
    sessions: Record<string, TerminalSession>;
    activeSessionId: string | null;
    config: TerminalConfig;
    isConnecting: boolean;
    error: AppError | null;
  }