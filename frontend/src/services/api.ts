// API Service for Container SSH System

import { Container, ContainerListResponse, TerminalSessionListResponse, ApiResponse } from '../types';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;

    const defaultOptions: RequestInit = {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
        ...options,
    };

    try {
        const response = await fetch(url, defaultOptions);

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP ${response.status}: ${errorText}`);
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.error(`API 요청 실패 (${endpoint}):`, error);
        throw error;
    }
}

// 컨테이너 관련 API
export const getContainers = async (): Promise<ContainerListResponse> => {
    return apiRequest<ContainerListResponse>('/api/containers');
};

export const getContainer = async (containerId: string): Promise<Container> => {
    return apiRequest<Container>(`/api/containers/${containerId}`);
};

// 터미널 세션 관련 API
export const getTerminalSessions = async (): Promise<TerminalSessionListResponse> => {
    return apiRequest<TerminalSessionListResponse>('/api/terminal/sessions');
};

// WebSocket 연결을 위한 URL 생성
export const getWebSocketUrl = (containerId: string): string => {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost = process.env.REACT_APP_WS_HOST || 'localhost:8080';
    return `${wsProtocol}//${wsHost}/ws/terminal/${containerId}`;
};

// 터미널 연결 상태 확인
export const checkContainerAccess = async (containerId: string): Promise<boolean> => {
    try {
        const container = await getContainer(containerId);
        return container.status === 'running' || container.status === 'online';
    } catch (error) {
        console.error('컨테이너 접근 확인 실패:', error);
        return false;
    }
};

// 헬스체크 API
export const healthCheck = async (): Promise<{ status: string; timestamp: string }> => {
    return apiRequest<{ status: string; timestamp: string}>('/api/health');
};

// 에러 처리를 위한 유틸리티 함수
export const handleApiError = (error: any): string => {
    if (error instanceof Error) {
        if (error.message.includes('Failed to fetch')) {
            return '서버에 연결할 수 없습니다. 네트워크 연결을 확인해주세요.';
        }
        if (error.message.includes('HTTP 404')) {
            return '요청한 리소스를 찾을 수 없습니다.';
        }
        if (error.message.includes('HTTP 403')) {
            return '접근 권한이 없습니다.';
        }
        if (error.message.includes('HTTP 500')) {
            return '서버 내부 오류가 발생했습니다.';
        }
        return error.message;
    }
    return '알 수 없는 오류가 발생했습니다.';
};

// 개발용 Mock 데이터 (아직 백엔드 준비X)
export const getMockContainers = (): ContainerListResponse => {
    return {
      containers: [
        {
          id: 'teleport-node-1',
          name: 'production-web',
          status: 'online',
          labels: {
            environment: 'production',
            service: 'web',
          },
          nodeAddr: '0.0.0.0:3022',
          image: 'nginx:latest',
          created: new Date().toISOString(),
          ports: ['80:8080', '443:8443'],
          uptime: '2d 5h 30m',
        },
        {
          id: 'teleport-node-2',
          name: 'database-primary',
          status: 'online',
          labels: {
            environment: 'production',
            service: 'database',
          },
          nodeAddr: '10.0.1.101:3022',
          image: 'postgres:13',
          created: new Date().toISOString(),
          ports: ['5432:5432'],
          uptime: '5d 12h 15m',
        },
        {
          id: 'teleport-node-3',
          name: 'staging-api',
          status: 'stopped',
          labels: {
            environment: 'staging',
            service: 'api',
          },
          nodeAddr: '10.0.2.100:3022',
          image: 'node:16-alpine',
          created: new Date().toISOString(),
          ports: ['3000:3000'],
          uptime: '0m',
        },
      ],
      total: 3,
    };
  };