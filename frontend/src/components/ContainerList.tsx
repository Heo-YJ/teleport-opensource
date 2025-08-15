// frontend/src/components/ContainerList.tsx
import React, { useState, useEffect } from 'react';
import { Container, TerminalSession } from '../types';
import { getContainers } from '../services/api';
import { WebTerminal } from './WebTerminal';
import './ContainerList.css';

const ContainerList: React.FC = () => {
  const [containers, setContainers] = useState<Container[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTerminals, setActiveTerminals] = useState<Record<string, TerminalSession>>({});

  useEffect(() => {
    loadContainers();
  }, []);

  const loadContainers = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getContainers();
      setContainers(response.containers);
    } catch (err) {
      setError('컨테이너 목록을 불러오는데 실패했습니다.');
      console.error('컨테이너 로딩 오류:', err);
    } finally {
      setLoading(false);
    }
  };

  const openTerminal = (container: Container) => {
    if (container.status !== 'running' && container.status !== 'online') {
      alert('실행 중인 컨테이너에만 접속할 수 있습니다.');
      return;
    }

    const terminalSession: TerminalSession = {
      id: `terminal-${container.id}-${Date.now()}`,
      containerId: container.id,
      containerName: container.name,
      status: 'connecting',
      createdAt: new Date().toISOString(),
    };

    setActiveTerminals(prev => ({
      ...prev,
      [terminalSession.id]: terminalSession
    }));
  };

  const closeTerminal = (sessionId: string) => {
    setActiveTerminals(prev => {
      const newTerminals = { ...prev };
      delete newTerminals[sessionId];
      return newTerminals;
    });
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
      case 'online':
        return '#4CAF50';
      case 'stopped':
        return '#F44336';
      case 'pending':
        return '#FF9800';
      case 'error':
        return '#E91E63';
      default:
        return '#9E9E9E';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'running':
      case 'online':
        return '실행 중';
      case 'stopped':
        return '중지됨';
      case 'pending':
        return '대기 중';
      case 'error':
        return '오류';
      default:
        return '알 수 없음';
    }
  };

  const canConnectToTerminal = (status: string) => {
    return status === 'running' || status === 'online';
  };

  if (loading) {
    return (
      <div className="container-list">
        <div className="loading">
          <div className="spinner"></div>
          <p>컨테이너 목록을 불러오는 중...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container-list">
        <div className="error">
          <p>{error}</p>
          <button onClick={loadContainers} className="retry-button">
            다시 시도
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="container-list">
      <div className="header">
        <h2>컨테이너 목록</h2>
        <div className="header-actions">
          <button onClick={loadContainers} className="refresh-button">
            새로고침
          </button>
          <span className="container-count">
            총 {containers.length}개 컨테이너
          </span>
        </div>
      </div>

      {containers.length === 0 ? (
        <div className="empty-state">
          <p>컨테이너가 없습니다.</p>
        </div>
      ) : (
        <div className="container-grid">
          {containers.map((container) => (
            <div key={container.id} className="container-card">
              <div className="container-header">
                <div className="container-info">
                  <h3 className="container-name">{container.name}</h3>
                  <span className="container-id">{container.id}</span>
                </div>
                <div className="container-status">
                  <div 
                    className="status-indicator"
                    style={{ backgroundColor: getStatusColor(container.status) }}
                  />
                  <span className="status-text">
                    {getStatusText(container.status)}
                  </span>
                </div>
              </div>

              <div className="container-details">
                <div className="detail-item">
                  <span className="label">노드 주소:</span>
                  <span className="value">{container.nodeAddr || 'N/A'}</span>
                </div>
                
                {container.image && (
                  <div className="detail-item">
                    <span className="label">이미지:</span>
                    <span className="value">{container.image}</span>
                  </div>
                )}

                {container.ports && container.ports.length > 0 && (
                  <div className="detail-item">
                    <span className="label">포트:</span>
                    <span className="value">{container.ports.join(', ')}</span>
                  </div>
                )}

                {container.uptime && (
                  <div className="detail-item">
                    <span className="label">가동 시간:</span>
                    <span className="value">{container.uptime}</span>
                  </div>
                )}
              </div>

              {container.labels && Object.keys(container.labels).length > 0 && (
                <div className="container-labels">
                  <span className="labels-title">라벨:</span>
                  <div className="labels-list">
                    {Object.entries(container.labels).map(([key, value]) => (
                      <span key={key} className="label-tag">
                        {key}: {value}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <div className="container-actions">
                <button
                  className={`terminal-button ${canConnectToTerminal(container.status) ? 'enabled' : 'disabled'}`}
                  onClick={() => openTerminal(container)}
                  disabled={!canConnectToTerminal(container.status)}
                  title={canConnectToTerminal(container.status) ? '터미널 열기' : '실행 중인 컨테이너만 접속 가능'}
                >
                  🖥️ 터미널
                </button>
                
                <button
                  className="details-button"
                  onClick={() => {/* TODO: 상세 정보 모달 */}}
                  title="상세 정보"
                >
                  📋 상세
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 활성 터미널들 */}
      {Object.values(activeTerminals).map((session) => (
        <div key={session.id} className="terminal-modal">
          <div className="terminal-container">
            <WebTerminal
              containerId={session.containerId}
              containerName={session.containerName}
              onClose={() => closeTerminal(session.id)}
            />
          </div>
        </div>
      ))}
    </div>
  );
};

export default ContainerList;