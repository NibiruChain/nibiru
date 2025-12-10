import React from 'react'
import type { AppConfig, PasskeyRegistration, ServiceHealth } from '../types'

interface SidebarProps {
  config: AppConfig
  passkey: PasskeyRegistration | null
  accountAddress: string
  balance: string
  isDeployed: boolean
  serviceHealth: { rpc: ServiceHealth; bundler: ServiceHealth }
  bundlerBalance: string
  bundlerLogs: { ts: number; level: string; message: string }[]
}

function displayHost(url: string): string {
  if (!url) return 'Not set'
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

function ServiceRow(props: { label: string; url: string; health: ServiceHealth }) {
  const tooltip = props.health.detail || props.url || props.label
  return (
    <div className="status-item">
      <span className="label">{props.label}</span>
      <div className="status-pill" title={tooltip}>
        <span className={`status-dot ${props.health.status}`} aria-label={`${props.label} ${props.health.status}`} />
        <span className="value truncate" title={props.url}>
          {displayHost(props.url)}
        </span>
      </div>
    </div>
  )
}

export function Sidebar({
  config,
  passkey,
  accountAddress,
  balance,
  isDeployed,
  serviceHealth,
  bundlerBalance,
  bundlerLogs
}: SidebarProps) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h3>State Monitor</h3>
      </div>

      <div className="sidebar-section">
        <h4>Connection</h4>
        <ServiceRow label="RPC" url={config.rpcUrl} health={serviceHealth.rpc} />
        <ServiceRow label="Bundler" url={config.bundlerUrl} health={serviceHealth.bundler} />
        <div className="status-item">
          <span className="label">Chain ID</span>
          <span className="value">{config.chainId}</span>
        </div>
        {config.bundlerSignerAddress && (
          <div className="status-item">
            <span className="label">Bundler signer</span>
            <span className="value truncate" title={config.bundlerSignerAddress}>
              {config.bundlerSignerAddress}
            </span>
          </div>
        )}
        {config.bundlerSignerAddress && (
          <div className="status-item">
            <span className="label">Bundler balance</span>
            <span className="value" title={`${bundlerBalance} wei`}>
              {Number(bundlerBalance) / 1e18} NIBI
            </span>
          </div>
        )}
      </div>

      <div className="sidebar-section">
        <h4>Passkey</h4>
        <div className="status-item">
          <span className="label">Status</span>
          <span className={`badge ${passkey ? 'success' : 'neutral'}`}>
            {passkey ? 'Registered' : 'Not Created'}
          </span>
        </div>
        {passkey && (
          <div className="status-item">
            <span className="label">Cred ID</span>
            <span className="value truncate" title={passkey.credentialId}>
              {passkey.credentialId}
            </span>
          </div>
        )}
      </div>

      <div className="sidebar-section">
        <h4>Account</h4>
        <div className="status-item">
          <span className="label">Status</span>
          <span className={`badge ${isDeployed ? 'success' : 'neutral'}`}>
            {isDeployed ? 'Deployed' : 'Not Deployed'}
          </span>
        </div>
        <div className="status-item">
          <span className="label">Address</span>
          <span className="value truncate" title={accountAddress}>
            {accountAddress}
          </span>
        </div>
        <div className="status-item">
          <span className="label">Balance</span>
          <span className="value">{Number(balance) / 1e18} NIBI</span>
        </div>
      </div>

      <div className="sidebar-section">
        <h4>Bundler Logs</h4>
        <div className="log-console" aria-live="polite">
          {bundlerLogs.length === 0 ? (
            <div className="log-line muted">No logs yet</div>
          ) : (
            bundlerLogs.map((log) => (
              <div key={`${log.ts}-${log.message.slice(0, 20)}`} className="log-line">
                <span className={`log-level ${log.level.toLowerCase()}`}>{log.level.toUpperCase()}</span>
                <span className="log-text">{log.message}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </aside>
  )
}
