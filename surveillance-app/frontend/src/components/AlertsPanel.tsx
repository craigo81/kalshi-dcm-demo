import { useState } from 'react';
import type { Alert, AlertSeverity } from '../types';

interface AlertsPanelProps {
  alerts: Alert[];
  onResolve: (id: string, notes: string) => void;
}

const severityBorderColors: Record<AlertSeverity, string> = {
  critical: 'border-red-500',
  high: 'border-orange-500',
  medium: 'border-yellow-500',
  low: 'border-blue-500',
};

const severityBadgeColors: Record<AlertSeverity, string> = {
  critical: 'bg-red-600',
  high: 'bg-orange-600',
  medium: 'bg-yellow-600',
  low: 'bg-blue-600',
};

export function AlertsPanel({ alerts, onResolve }: AlertsPanelProps) {
  const [filter, setFilter] = useState<AlertSeverity | ''>('');

  const filteredAlerts = filter
    ? alerts.filter((a) => a.severity === filter)
    : alerts;

  const handleResolve = (id: string) => {
    const notes = window.prompt('Resolution notes:');
    if (notes !== null) {
      onResolve(id, notes);
    }
  };

  return (
    <div className="bg-dark-800 rounded-lg border border-dark-700">
      <div className="px-4 py-3 border-b border-dark-700 flex justify-between items-center">
        <h2 className="font-semibold">Active Alerts</h2>
        <select
          value={filter}
          onChange={(e) => setFilter(e.target.value as AlertSeverity | '')}
          className="bg-dark-700 text-sm rounded px-2 py-1 border-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">All Severities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      </div>
      <div className="p-4 space-y-3 max-h-96 overflow-y-auto">
        {filteredAlerts.length === 0 ? (
          <p className="text-gray-500 text-center py-4">No open alerts</p>
        ) : (
          filteredAlerts.map((alert) => (
            <div
              key={alert.id}
              className={`bg-dark-700 rounded p-3 border-l-4 ${
                severityBorderColors[alert.severity]
              }`}
            >
              <div className="flex justify-between items-start">
                <div>
                  <span
                    className={`inline-block px-2 py-0.5 text-xs rounded ${
                      severityBadgeColors[alert.severity]
                    }`}
                  >
                    {alert.severity.toUpperCase()}
                  </span>
                  <span className="text-sm text-gray-400 ml-2">{alert.type}</span>
                </div>
                <button
                  onClick={() => handleResolve(alert.id)}
                  className="text-xs text-blue-400 hover:text-blue-300 transition"
                >
                  Resolve
                </button>
              </div>
              <p className="mt-2 text-sm">{alert.description}</p>
              <div className="mt-2 text-xs text-gray-500">
                <span>User: {alert.user_id}</span>
                <span className="mx-2">|</span>
                <span>Market: {alert.market_ticker}</span>
                <span className="mx-2">|</span>
                <span>{new Date(alert.created_at).toLocaleString()}</span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
