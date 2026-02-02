import type { ActivityLogEntry } from '../types';

interface ActivityFeedProps {
  entries: ActivityLogEntry[];
  isConnected: boolean;
}

const typeColors: Record<ActivityLogEntry['type'], string> = {
  info: 'text-blue-400',
  success: 'text-green-400',
  warning: 'text-yellow-400',
  critical: 'text-red-400',
};

export function ActivityFeed({ entries, isConnected }: ActivityFeedProps) {
  return (
    <div className="bg-dark-800 rounded-lg border border-dark-700 mt-6">
      <div className="px-4 py-3 border-b border-dark-700 flex justify-between items-center">
        <h2 className="font-semibold">Live Activity Feed</h2>
        <div className="flex items-center space-x-2">
          <span
            className={`w-2 h-2 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          <span className="text-sm text-gray-400">
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>
      <div className="p-4 space-y-2 max-h-48 overflow-y-auto font-mono text-sm">
        {entries.length === 0 ? (
          <p className="text-gray-500">
            {isConnected ? 'Waiting for activity...' : 'Connecting to WebSocket...'}
          </p>
        ) : (
          entries.map((entry) => (
            <div key={entry.id} className={typeColors[entry.type]}>
              <span className="text-gray-500">
                [{entry.timestamp.toLocaleTimeString()}]
              </span>{' '}
              {entry.message}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
