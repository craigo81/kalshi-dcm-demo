import type { Stats } from '../types';

interface HeaderProps {
  stats: Stats | null;
  isWsConnected: boolean;
}

export function Header({ stats, isWsConnected }: HeaderProps) {
  const getStatusColor = () => {
    if (!stats) return 'bg-gray-500';
    switch (stats.system_status) {
      case 'halted':
        return 'bg-red-500';
      case 'warning':
        return 'bg-yellow-500';
      default:
        return 'bg-green-500';
    }
  };

  const getStatusText = () => {
    if (!stats) return 'Loading...';
    switch (stats.system_status) {
      case 'halted':
        return 'System Halted';
      case 'warning':
        return 'Warning';
      default:
        return 'System Operational';
    }
  };

  const formatLastUpdated = () => {
    if (!stats?.last_updated) return '--';
    return 'Updated: ' + new Date(stats.last_updated).toLocaleTimeString();
  };

  return (
    <header className="bg-dark-800 border-b border-dark-700 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className="flex items-center">
            <span className="text-2xl font-bold text-blue-400">🔍</span>
            <h1 className="ml-2 text-xl font-bold">DCM Surveillance Dashboard</h1>
          </div>
          <span className="px-2 py-1 text-xs bg-blue-600 rounded">CFTC CP 4</span>
        </div>
        <div className="flex items-center space-x-6">
          <div className="flex items-center space-x-2">
            <span
              className={`w-3 h-3 rounded-full animate-pulse-slow ${getStatusColor()}`}
            />
            <span className="text-sm text-gray-400">{getStatusText()}</span>
          </div>
          <div className="flex items-center space-x-2">
            <span
              className={`w-2 h-2 rounded-full ${
                isWsConnected ? 'bg-green-500' : 'bg-red-500'
              }`}
            />
            <span className="text-xs text-gray-500">
              {isWsConnected ? 'Live' : 'Disconnected'}
            </span>
          </div>
          <span className="text-sm text-gray-500">{formatLastUpdated()}</span>
        </div>
      </div>
    </header>
  );
}
