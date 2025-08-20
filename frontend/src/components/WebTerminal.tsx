import React, { useEffect, useRef, useState } from 'react';
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
  const [output, setOutput] = useState<string[]>([]);
  const [input, setInput] = useState<string>('');
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting');
  const socket = useRef<WebSocket | null>(null);
  const outputRef = useRef<HTMLDivElement>(null);
  const isConnecting = useRef<boolean>(false);

  useEffect(() => {
    // 중복 연결 방지
    if (isConnecting.current) {
      console.log('⚠️ 이미 연결 중입니다. 중복 연결 방지');
      return;
    }

    connectWebSocket();
    
    return () => {
      if (socket.current) {
        socket.current.close();
      }
    };
  }, [containerId]);

  // 출력이 업데이트될 때마다 스크롤을 맨 아래로
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [output]);

  const connectWebSocket = () => {
    const wsUrl = `ws://localhost:8080/api/ws/terminal/${containerId}`;
    console.log('🔗 WebSocket 연결 시도:', wsUrl);

    socket.current = new WebSocket(wsUrl);

    socket.current.onopen = () => {
      console.log('✅ WebSocket 연결 성공!');
      setConnectionStatus('connected');
      addOutput('✅ 터미널 연결이 설정되었습니다.', 'system');
      addOutput(`🐳 컨테이너: ${containerName} (${containerId})`, 'system');
      addOutput('명령어를 입력하세요...', 'system');
    };

    socket.current.onmessage = (event) => {
      console.log('📨 메시지 받음:', event.data);
      
      try {
        const message = JSON.parse(event.data);
        console.log('📋 파싱된 메시지:', message);
        
        switch (message.type) {
          case 'output':
            addOutput(message.data, 'output');
            break;
          case 'system':
            const systemMsg = typeof message.data === 'string' ? message.data : message.data?.message || '시스템 메시지';
            addOutput(`[시스템] ${systemMsg}`, 'system');
            break;
          case 'error':
            const errorMsg = typeof message.data === 'string' ? message.data : message.data?.message || '에러 발생';
            addOutput(`[에러] ${errorMsg}`, 'error');
            break;
          case 'exit':
            const exitMsg = message.data?.message || '터미널 세션이 종료되었습니다';
            addOutput(`[종료] ${exitMsg}`, 'system');
            setConnectionStatus('disconnected');
            break;
          case 'pong':
            console.log('🏓 Pong 받음:', message.data);
            addOutput('🏓 서버 응답: Pong', 'system');
            break;
          default:
            console.log('❓ 알 수 없는 메시지 타입:', message);
            if (message.data) {
              addOutput(message.data, 'output');
            }
        }
      } catch (e) {
        // JSON이 아닌 원시 데이터인 경우
        console.log('📄 원시 데이터:', event.data);
        addOutput(event.data, 'output');
      }
    };

    socket.current.onclose = (event) => {
      console.log('🔌 WebSocket 연결 종료:', event.code, event.reason);
      setConnectionStatus('disconnected');
      addOutput('🔌 연결이 종료되었습니다.', 'error');
    };

    socket.current.onerror = (error) => {
      console.error('❌ WebSocket 오류:', error);
      setConnectionStatus('error');
      addOutput('❌ 연결 오류가 발생했습니다.', 'error');
    };
  };

  const addOutput = (text: string, type: 'output' | 'system' | 'error' = 'output') => {
    const timestamp = new Date().toLocaleTimeString();
    const formattedText = `[${timestamp}] ${text}`;
    
    setOutput(prev => [...prev, formattedText]);
  };

  const sendCommand = () => {
    if (!input.trim() || !socket.current || socket.current.readyState !== WebSocket.OPEN) {
      return;
    }

    const command = input.trim();
    console.log('⌨️ 명령어 전송:', command);

    // 사용자 입력을 화면에 표시
    addOutput(`$ ${command}`, 'output');

    // 백엔드로 전송
    const message = {
      type: 'input',
      data: command + '\n'  // Enter 추가
    };
    
    console.log('📤 전송할 메시지:', message);
    socket.current.send(JSON.stringify(message));
    
    setInput('');
  };

  const sendPing = () => {
    if (socket.current && socket.current.readyState === WebSocket.OPEN) {
      const pingMessage = {
        type: 'ping',
        data: 'ping test'
      };
      console.log('🏓 Ping 전송:', pingMessage);
      socket.current.send(JSON.stringify(pingMessage));
      addOutput('🏓 Ping 전송됨', 'system');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      sendCommand();
    }
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
          {connectionStatus === 'connected' && (
            <button 
              onClick={sendPing}
              title="Ping 테스트"
              style={{ 
                marginRight: '10px', 
                padding: '5px 10px', 
                fontSize: '12px',
                backgroundColor: '#007acc',
                color: 'white',
                border: 'none',
                borderRadius: '3px',
                cursor: 'pointer'
              }}
            >
              🏓 Ping
            </button>
          )}
          <button 
            className="close-button" 
            onClick={onClose}
            title="터미널 종료"
          >
            ✕
          </button>
        </div>
      </div>
      
      {/* 터미널 출력 영역 */}
      <div 
        className="terminal-output"
        ref={outputRef}
        style={{
          backgroundColor: '#1e1e1e',
          color: '#ffffff',
          fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
          fontSize: '14px',
          padding: '15px',
          height: '400px',
          overflowY: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word'
        }}
      >
        {output.map((line, index) => (
          <div key={index} style={{ marginBottom: '2px' }}>
            {line}
          </div>
        ))}
      </div>
      
      {/* 입력 영역 */}
      <div 
        className="terminal-input"
        style={{
          display: 'flex',
          padding: '10px',
          backgroundColor: '#2d2d2d',
          borderTop: '1px solid #555'
        }}
      >
        <span style={{ color: '#4CAF50', marginRight: '10px' }}>$</span>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="명령어를 입력하세요..."
          disabled={connectionStatus !== 'connected'}
          style={{
            flex: 1,
            backgroundColor: 'transparent',
            border: 'none',
            color: '#ffffff',
            fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
            fontSize: '14px',
            outline: 'none'
          }}
        />
        <button
          onClick={sendCommand}
          disabled={!input.trim() || connectionStatus !== 'connected'}
          style={{
            marginLeft: '10px',
            padding: '5px 15px',
            backgroundColor: '#4CAF50',
            color: 'white',
            border: 'none',
            borderRadius: '3px',
            cursor: connectionStatus === 'connected' ? 'pointer' : 'not-allowed',
            opacity: connectionStatus === 'connected' ? 1 : 0.5
          }}
        >
          전송
        </button>
      </div>
    </div>
  );
};