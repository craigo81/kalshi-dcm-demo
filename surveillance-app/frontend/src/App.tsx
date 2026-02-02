import { useState, useEffect, useCallback } from 'react';
import {
  Header,
  StatsGrid,
  EmergencyControls,
  AlertsPanel,
  MarketsPanel,
  UsersTable,
  ActivityFeed,
} from './components';
import { useWebSocket } from './hooks/useWebSocket';
import * as api from './api/client';
import type {
  Stats,
  Alert,
  Market,
  SurveillanceUser,
  ActivityLogEntry,
  WSMessage,
  InitialStateData,
} from './types';

const OPERATOR_EMAIL = 'operator@dcm.com';

function App() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [markets, setMarkets] = useState<Market[]>([]);
  const [users, setUsers] = useState<SurveillanceUser[]>([]);
  const [activityLog, setActivityLog] = useState<ActivityLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // WebSocket connection
  const wsUrl = `ws://${window.location.host}/ws`;
  const { isConnected, lastMessage } = useWebSocket(wsUrl);

  // Add activity log entry
  const addActivity = useCallback(
    (message: string, type: ActivityLogEntry['type'] = 'info') => {
      setActivityLog((prev) => {
        const newEntry: ActivityLogEntry = {
          id: crypto.randomUUID(),
          timestamp: new Date(),
          message,
          type,
        };
        // Keep only last 50 entries
        return [newEntry, ...prev].slice(0, 50);
      });
    },
    []
  );

  // Fetch all data
  const fetchData = useCallback(async () => {
    try {
      const [statsData, alertsData, marketsData, usersData] = await Promise.all([
        api.fetchStats(),
        api.fetchAlerts('open'),
        api.fetchMarkets(),
        api.fetchUsers(),
      ]);
      setStats(statsData);
      setAlerts(alertsData);
      setMarkets(marketsData);
      setUsers(usersData);
      setIsLoading(false);
    } catch (err) {
      console.error('Failed to fetch data:', err);
      addActivity('Failed to fetch data', 'critical');
    }
  }, [addActivity]);

  // Initial data load
  useEffect(() => {
    fetchData();
    // Refresh every 30 seconds as backup
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // Handle WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    const { type, data } = lastMessage as WSMessage;

    switch (type) {
      case 'initial_state': {
        const state = data as InitialStateData;
        setStats(state.stats);
        setAlerts(state.alerts.filter((a) => a.status === 'open'));
        setMarkets(state.markets);
        addActivity('WebSocket connected', 'info');
        break;
      }
      case 'stats_update':
        setStats(data as Stats);
        break;
      case 'alert_resolved': {
        const resolved = data as { id: string };
        setAlerts((prev) => prev.filter((a) => a.id !== resolved.id));
        addActivity(`Alert ${resolved.id} resolved`, 'success');
        break;
      }
      case 'new_alert': {
        const newAlert = data as Alert;
        setAlerts((prev) => [newAlert, ...prev]);
        addActivity(`New ${newAlert.severity} alert: ${newAlert.type}`, 'warning');
        break;
      }
      case 'market_halted': {
        const halted = data as { ticker: string; reason: string };
        addActivity(`Market ${halted.ticker} HALTED: ${halted.reason}`, 'critical');
        fetchData();
        break;
      }
      case 'market_resumed': {
        const resumed = data as { ticker: string };
        addActivity(`Market ${resumed.ticker} resumed trading`, 'success');
        fetchData();
        break;
      }
      case 'global_halt': {
        const halt = data as { reason: string };
        addActivity(`🛑 GLOBAL HALT: ${halt.reason}`, 'critical');
        fetchData();
        break;
      }
      case 'global_resume':
        addActivity(`▶️ Global trading resumed`, 'success');
        fetchData();
        break;
      case 'user_suspended': {
        const suspended = data as { email: string };
        addActivity(`User ${suspended.email} suspended`, 'warning');
        fetchData();
        break;
      }
    }
  }, [lastMessage, fetchData, addActivity]);

  // Handlers
  const handleResolveAlert = async (id: string, notes: string) => {
    try {
      await api.resolveAlert(id, OPERATOR_EMAIL, notes);
      setAlerts((prev) => prev.filter((a) => a.id !== id));
      addActivity(`Alert ${id} resolved`, 'success');
    } catch (err) {
      console.error('Failed to resolve alert:', err);
      addActivity('Failed to resolve alert', 'critical');
    }
  };

  const handleHaltMarket = async (ticker: string, reason: string) => {
    try {
      await api.haltMarket(ticker, reason, OPERATOR_EMAIL);
      addActivity(`Market ${ticker} halt initiated`, 'warning');
      await fetchData();
    } catch (err) {
      console.error('Failed to halt market:', err);
      addActivity('Failed to halt market', 'critical');
    }
  };

  const handleResumeMarket = async (ticker: string) => {
    try {
      await api.resumeMarket(ticker);
      addActivity(`Market ${ticker} resume initiated`, 'info');
      await fetchData();
    } catch (err) {
      console.error('Failed to resume market:', err);
      addActivity('Failed to resume market', 'critical');
    }
  };

  const handleSuspendUser = async (id: string) => {
    try {
      await api.suspendUser(id);
      addActivity(`User ${id} suspended`, 'warning');
      await fetchData();
    } catch (err) {
      console.error('Failed to suspend user:', err);
      addActivity('Failed to suspend user', 'critical');
    }
  };

  const handleGlobalHalt = async () => {
    const reason = window.prompt('Global halt reason:');
    if (!reason) return;

    if (!window.confirm('⚠️ This will halt ALL trading. Are you sure?')) return;

    try {
      await api.globalHalt(reason, OPERATOR_EMAIL);
      addActivity(`Global halt initiated: ${reason}`, 'critical');
      await fetchData();
    } catch (err) {
      console.error('Failed to initiate global halt:', err);
      addActivity('Failed to initiate global halt', 'critical');
    }
  };

  const handleGlobalResume = async () => {
    if (!window.confirm('Resume all trading?')) return;

    try {
      await api.globalResume();
      addActivity('Global trading resumed', 'success');
      await fetchData();
    } catch (err) {
      console.error('Failed to resume trading:', err);
      addActivity('Failed to resume trading', 'critical');
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-dark-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto" />
          <p className="mt-4 text-gray-400">Loading surveillance dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-dark-900 text-white">
      <Header stats={stats} isWsConnected={isConnected} />

      <main className="p-6">
        <StatsGrid stats={stats} />

        <EmergencyControls
          onGlobalHalt={handleGlobalHalt}
          onGlobalResume={handleGlobalResume}
          isHalted={stats?.system_status === 'halted'}
        />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <AlertsPanel alerts={alerts} onResolve={handleResolveAlert} />
          <MarketsPanel
            markets={markets}
            onHalt={handleHaltMarket}
            onResume={handleResumeMarket}
          />
        </div>

        <UsersTable users={users} onSuspend={handleSuspendUser} />

        <ActivityFeed entries={activityLog} isConnected={isConnected} />
      </main>
    </div>
  );
}

export default App;
