import React, { useState, useEffect } from 'react';
import ContainerList from './components/ContainerList';
import { healthCheck } from './services/api';
import './App.css';


interface AppState {
    isServerConnected: boolean;
    isCheckingConnection: boolean;
    lastConnectionCheck: string | null;
}

function App() {
  const [appState, setAppState] = useState<AppState>({
    isServerConnected: false,
    isCheckingConnection: true,
    lastConnectionCheck: null,
  });

  useEffect(() => {
    checkServerConnection();

    // 주기적으로 서버 연결 상태 확인 (30초마다)
    const interval = setInterval(checkServerConnection, 30000);

    return () => clearInterval(interval);
  }, []);

  // 서버 연결 상태 실시간 모니터링
  const checkServerConnection = async () => {
    setAppState(prev => ({ ...prev, isCheckingConnection: true}));

    try {
        await healthCheck();
        setAppState({
            isServerConnected: true,
            isCheckingConnection: false,
            lastConnectionCheck: new Date().toLocaleTimeString(),
        });
    } catch (error) {
        console.error('서버 연결 확인 실패:', error);
        setAppState({
            isServerConnected: false,
            isCheckingConnection: false,
            lastConnectionCheck: new Date().toLocaleTimeString(),
        });
    }
  };

  const ConnectionStatus: React.FC = () => (
    <div className={`connection-status ${appState.isServerConnected ? 'connected' : 'disconnected'}`}>
      <div className="status-indicator">
        <div className={`status-dot ${appState.isServerConnected ? 'online' : 'offline'}`} />
        <span className="status-text">
          {appState.isCheckingConnection 
            ? '연결 확인 중...' 
            : appState.isServerConnected 
              ? '서버 연결됨' 
              : '서버 연결 안됨'
          }
        </span>
      </div>
      {appState.lastConnectionCheck && (
        <span className="last-check">
          마지막 확인: {appState.lastConnectionCheck}
        </span>
      )}
      <button 
        onClick={checkServerConnection} 
        className="refresh-connection"
        disabled={appState.isCheckingConnection}
        title="연결 상태 새로고침"
      >
        🔄
      </button>
    </div>
  );

  return (
    <div className="App">
      <header className="app-header">
        <div className="header-content">
          <div className="title-section">
            <h1>🐳 Container SSH Manager</h1>
            <p className="subtitle">Teleport 기반 웹 컨테이너 터미널 접속 시스템</p>
          </div>
          <ConnectionStatus />
        </div>
      </header>

      <main className="app-main">
        {!appState.isServerConnected && !appState.isCheckingConnection ? (
          <div className="server-error">
            <div className="error-content">
              <h2>🚫 서버 연결 실패</h2>
              <p>백엔드 서버에 연결할 수 없습니다.</p>
              <div className="error-details">
                <p>다음 사항을 확인해주세요:</p>
                <ul>
                  <li>백엔드 서버가 실행 중인지 확인 (포트 8080)</li>
                  <li>네트워크 연결 상태 확인</li>
                  <li>CORS 설정 확인</li>
                </ul>
              </div>
              <button onClick={checkServerConnection} className="retry-button">
                다시 시도
              </button>
            </div>
          </div>
        ) : (
          <ContainerList />
        )}
      </main>

      <footer className="app-footer">
        <div className="footer-content">
          <p>&copy; 2025 Container SSH Manager. Powered by Teleport & React.</p>
          <div className="footer-links">
            <span>버전: 1.0.0</span>
            <span>•</span>
            <a href="#" onClick={(e) => { e.preventDefault(); /* TODO: 도움말 */ }}>
              도움말
            </a>
            <span>•</span>
            <a href="#" onClick={(e) => { e.preventDefault(); /* TODO: 설정 */ }}>
              설정
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;