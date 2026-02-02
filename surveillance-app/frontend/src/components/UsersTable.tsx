import type { SurveillanceUser, UserStatus } from '../types';

interface UsersTableProps {
  users: SurveillanceUser[];
  onSuspend: (id: string) => void;
}

const statusBadgeColors: Record<UserStatus, string> = {
  verified: 'bg-green-600',
  kyc_pending: 'bg-yellow-600',
  suspended: 'bg-red-600',
  banned: 'bg-gray-600',
};

export function UsersTable({ users, onSuspend }: UsersTableProps) {
  const handleSuspend = (id: string) => {
    if (window.confirm('Are you sure you want to suspend this user?')) {
      onSuspend(id);
    }
  };

  const getUtilizationClass = (utilization: number) => {
    if (utilization > 90) return 'text-red-400';
    if (utilization > 70) return 'text-yellow-400';
    return 'text-green-400';
  };

  return (
    <div className="bg-dark-800 rounded-lg border border-dark-700 mt-6">
      <div className="px-4 py-3 border-b border-dark-700">
        <h2 className="font-semibold">User Surveillance</h2>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-dark-700">
            <tr>
              <th className="text-left px-4 py-2 text-sm text-gray-400">User</th>
              <th className="text-left px-4 py-2 text-sm text-gray-400">Status</th>
              <th className="text-right px-4 py-2 text-sm text-gray-400">
                Exposure
              </th>
              <th className="text-right px-4 py-2 text-sm text-gray-400">Limit</th>
              <th className="text-right px-4 py-2 text-sm text-gray-400">
                Utilization
              </th>
              <th className="text-right px-4 py-2 text-sm text-gray-400">Alerts</th>
              <th className="text-center px-4 py-2 text-sm text-gray-400">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {users.length === 0 ? (
              <tr>
                <td colSpan={7} className="text-center py-4 text-gray-500">
                  No users to display
                </td>
              </tr>
            ) : (
              users.map((user) => {
                const utilization =
                  (user.current_exposure / user.position_limit) * 100;

                return (
                  <tr key={user.id} className="border-t border-dark-700">
                    <td className="px-4 py-3">
                      <div className="font-medium">{user.email}</div>
                      <div className="text-xs text-gray-500">{user.id}</div>
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 py-0.5 text-xs rounded ${
                          statusBadgeColors[user.status]
                        }`}
                      >
                        {user.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right font-mono">
                      ${user.current_exposure.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right font-mono">
                      ${user.position_limit.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span className={getUtilizationClass(utilization)}>
                        {utilization.toFixed(1)}%
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      {user.alert_count > 0 ? (
                        <span className="px-2 py-0.5 text-xs bg-red-600 rounded">
                          {user.alert_count}
                        </span>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {user.status !== 'suspended' ? (
                        <button
                          onClick={() => handleSuspend(user.id)}
                          className="text-xs text-red-400 hover:text-red-300 transition"
                        >
                          Suspend
                        </button>
                      ) : (
                        <span className="text-xs text-gray-500">Suspended</span>
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
