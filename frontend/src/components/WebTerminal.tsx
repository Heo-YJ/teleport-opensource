// frontend/src/components/WebTerminal.tsx
import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import 'xterm/css/xterm.css';
import './WebTerminal.css';

interface WebTerminalProps {
  containerId: string;
  containerName: string;
  onClose: () => void;
}

export const WebTerminal: React.FC<WebTerminalProps> = ({ 
  containerId, 
  containerName, 
  onClose 
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const socket = useRef<WebSocket | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting');

  useEffect(() => {
    if (!terminalRef.current) return;

    // 터미널 인스턴스 생성
    terminal.current = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
        cursor: '#ffffff',
        selectionBackground: '#ffffff40',
      },
      rows: 24,
      cols: 80,
    });

    // 애드온 추가
    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);
    terminal.current.loadAddon(new WebLinksAddon());

    // 터미널을 DOM에 연결
    terminal.current.open(terminalRef.current);
    fitAddon.current.fit();

    // WebSocket 연결 설정
    connectWebSocket();

    // 창 크기 변경 시 터미널 크기 조정
    const handleResize = () => {
      if (fitAddon.current) {
        fitAddon.current.fit();
      }
    };
    window.addEventListener('resize', handleResize);

    // 정리 함수
    return () => {
      window.removeEventListener('resize', handleResize);
      if (socket.current) {
        socket.current.close();
      }
      if (terminal.current) {
        terminal.current.dispose();
      }
    };
  }, [containerId]);

  const connectWebSocket = () => {
    const wsUrl = `ws://localhost:8080/ws/terminal/${containerId}`;
    socket.current = new WebSocket(wsUrl);

    socket.current.onopen = () => {
      console.log('WebSocket 연결됨');
      setConnectionStatus('connected');
      
      if (terminal.current) {
        terminal.current.writeln('\x1b[32m터미널 연결이 설정되었습니다.\x1b[0m');
        terminal.current.writeln(`컨테이너: ${containerName} (${containerId})`);
        terminal.current.write('\r\n$ ');
      }
    };

    socket.current.onmessage = (event) => {
      if (terminal.current) {
        terminal.current.write(event.data);
      }
    };

    socket.current.onclose = (event) => {
      console.log('WebSocket 연결 종료:', event.code, event.reason);
      setConnectionStatus('disconnected');
      
      if (terminal.current) {
        terminal.current.writeln('\r\n\x1b[31m연결이 종료되었습니다.\x1b[0m');
      }
    };

    socket.current.onerror = (error) => {
      console.error('WebSocket 오류:', error);
      setConnectionStatus('error');
      
      if (terminal.current) {
        terminal.current.writeln('\r\n\x1b[31m연결 오류가 발생했습니다.\x1b[0m');
      }
    };

    // 터미널 입력을 WebSocket으로 전송
    if (terminal.current) {
      terminal.current.onData((data) => {
        if (socket.current && socket.current.readyState === WebSocket.OPEN) {
          socket.current.send(JSON.stringify({
            type: 'input',
            data: data
          }));
        }
      });
    }
  };

  const handleDisconnect = () => {
    if (socket.current) {
      socket.current.close();
    }
    onClose();
  };

  const getStatusColor = () => {
    switch (connectionStatus) {
      case 'connected': return '#4CAF50';
      case 'connecting': return '#FF9800';
      case 'disconnected': return '#9E9E9E';
      case 'error': return '#F44336';
      default: return '#9E9E9E';
    }
  };

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected': return '연결됨';
      case 'connecting': return '연결 중...';
      case 'disconnected': return '연결 끊어짐';
      case 'error': return '오류';
      default: return '알 수 없음';
    }
  };

  return (
    <div className="web-terminal">
      <div className="terminal-header">
        <div className="terminal-title">
          <span className="container-name">{containerName}</span>
          <span className="container-id">({containerId})</span>
        </div>
        <div className="terminal-controls">
          <div className="connection-status">
            <div 
              className="status-indicator" 
              style={{ backgroundColor: getStatusColor() }}
            />
            <span className="status-text">{getStatusText()}</span>
          </div>
          <button 
            className="close-button" 
            onClick={handleDisconnect}
            title="터미널 종료"
          >
            ✕
          </button>
        </div>
      </div>
      <div className="terminal-content" ref={terminalRef} />
    </div>
  );
};