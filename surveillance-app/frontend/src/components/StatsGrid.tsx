import type { Stats } from '../types';

interface StatsGridProps {
  stats: Stats | null;
}

interface StatCardProps {
  label: string;
  value: string | number;
  icon: string;
  subValue?: string;
  subValueClassName?: string;
}

function StatCard({ label, value, icon, subValue, subValueClassName }: StatCardProps) {
  return (
    <div className="bg-dark-800 rounded-lg p-4 border border-dark-700">
      <div className="flex items-center justify-between">
        <span className="text-gray-400 text-sm">{label}</span>
        <span className="text-2xl">{icon}</span>
      </div>
      <p className="text-3xl font-bold mt-2">{value}</p>
      {subValue && (
        <p className={`text-sm mt-1 ${subValueClassName || 'text-gray-400'}`}>
          {subValue}
        </p>
      )}
    </div>
  );
}

export function StatsGrid({ stats }: StatsGridProps) {
  const formatVolume = (volume: number) => {
    return '$' + volume.toLocaleString();
  };

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <StatCard
        label="Active Users"
        value={stats?.active_users ?? '--'}
        icon="👥"
      />
      <StatCard
        label="Open Positions"
        value={stats?.open_positions ?? '--'}
        icon="📈"
      />
      <StatCard
        label="Open Alerts"
        value={stats?.open_alerts ?? '--'}
        icon="⚠️"
        subValue={`${stats?.critical_alerts ?? '--'} critical`}
        subValueClassName="text-red-400"
      />
      <StatCard
        label="24h Volume"
        value={stats ? formatVolume(stats.total_volume_24h) : '--'}
        icon="💰"
      />
    </div>
  );
}
