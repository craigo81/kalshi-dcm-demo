// Positions Panel Component
// Core Principle 5: Position monitoring and limits

import React, { useState, useEffect } from 'react';
import { Briefcase, TrendingUp, TrendingDown, RefreshCw, Loader2, AlertTriangle, Gauge } from 'lucide-react';
import { portfolioAPI, Position, PortfolioSummary } from '../../api/client';

// Progress bar for position limit utilization
function UtilizationBar({ utilization }: { utilization: number }) {
  const getColor = () => {
    if (utilization >= 90) return 'bg-red-500';
    if (utilization >= 70) return 'bg-amber-500';
    return 'bg-emerald-500';
  };

  const getTextColor = () => {
    if (utilization >= 90) return 'text-red-400';
    if (utilization >= 70) return 'text-amber-400';
    return 'text-emerald-400';
  };

  return (
    <div className="space-y-2">
      <div className="flex justify-between items-center text-xs">
        <span className="text-slate-400">Position Limit Utilization</span>
        <span className={`font-medium ${getTextColor()}`}>{utilization.toFixed(1)}%</span>
      </div>
      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full ${getColor()} transition-all duration-300`}
          style={{ width: `${Math.min(100, utilization)}%` }}
        />
      </div>
      {utilization >= 90 && (
        <div className="flex items-center gap-1 text-xs text-red-400">
          <AlertTriangle className="w-3 h-3" />
          <span>Approaching position limit</span>
        </div>
      )}
    </div>
  );
}

export function PositionsPanel() {
  const [positions, setPositions] = useState<Position[]>([]);
  const [summary, setSummary] = useState<PortfolioSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchData = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const [posData, summaryData] = await Promise.all([
        portfolioAPI.getPositions(),
        portfolioAPI.getSummary(),
      ]);
      setPositions(posData.positions || []);
      setSummary(summaryData);
    } catch (error) {
      console.error('Failed to fetch portfolio data:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(() => fetchData(true), 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="card">
        <div className="flex items-center justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-primary-500" />
        </div>
      </div>
    );
  }

  const totalValue = summary?.positions.total_value || 0;
  const totalPnL = summary?.positions.unrealized_pnl || 0;
  const utilization = summary?.limits.utilization || 0;

  return (
    <div className="card">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-primary-500/20 rounded-lg flex items-center justify-center">
            <Briefcase className="w-5 h-5 text-primary-400" />
          </div>
          <div>
            <h3 className="font-semibold text-white">Open Positions</h3>
            <p className="text-xs text-slate-400">Core Principle 5: Position Monitoring</p>
          </div>
        </div>
        <button
          onClick={() => fetchData(true)}
          disabled={refreshing}
          className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
        >
          <RefreshCw className={`w-4 h-4 text-slate-400 ${refreshing ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Position Limit Utilization */}
      {summary && (
        <div className="p-4 bg-slate-700/30 rounded-lg mb-4">
          <UtilizationBar utilization={utilization} />
          <div className="flex justify-between text-xs text-slate-500 mt-2">
            <span>Exposure: ${summary.limits.current_exposure.toFixed(2)}</span>
            <span>Limit: ${summary.limits.position_limit.toFixed(2)}</span>
          </div>
        </div>
      )}

      {/* Summary */}
      <div className="grid grid-cols-2 gap-4 mb-6">
        <div className="p-4 bg-slate-700/30 rounded-lg">
          <p className="text-xs text-slate-400 mb-1">Total Value</p>
          <p className="text-xl font-bold text-white">${totalValue.toFixed(2)}</p>
        </div>
        <div className="p-4 bg-slate-700/30 rounded-lg">
          <p className="text-xs text-slate-400 mb-1">Unrealized P&L</p>
          <p className={`text-xl font-bold ${totalPnL >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
            {totalPnL >= 0 ? '+' : ''}${totalPnL.toFixed(2)}
          </p>
        </div>
      </div>

      {/* Positions List */}
      {positions.length === 0 ? (
        <div className="text-center py-8">
          <Briefcase className="w-12 h-12 text-slate-600 mx-auto mb-3" />
          <p className="text-slate-400">No open positions</p>
          <p className="text-sm text-slate-500 mt-1">Your active trades will appear here</p>
        </div>
      ) : (
        <div className="space-y-3">
          {positions.map((position) => (
            <PositionRow key={position.id} position={position} />
          ))}
        </div>
      )}
    </div>
  );
}

function PositionRow({ position }: { position: Position }) {
  const isProfit = position.unrealized_pnl_usd >= 0;

  return (
    <div className="p-4 bg-slate-700/30 rounded-lg">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {position.side === 'yes' ? (
            <TrendingUp className="w-4 h-4 text-emerald-400" />
          ) : (
            <TrendingDown className="w-4 h-4 text-red-400" />
          )}
          <span className="font-medium text-white">{position.market_ticker}</span>
          <span className={`badge ${position.side === 'yes' ? 'badge-success' : 'badge-danger'}`}>
            {position.side.toUpperCase()}
          </span>
        </div>
        <span className={`font-semibold ${isProfit ? 'text-emerald-400' : 'text-red-400'}`}>
          {isProfit ? '+' : ''}{position.unrealized_pnl_usd.toFixed(2)}
        </span>
      </div>

      <div className="grid grid-cols-4 gap-2 text-sm">
        <div>
          <p className="text-slate-500">Qty</p>
          <p className="text-slate-300">{position.quantity}</p>
        </div>
        <div>
          <p className="text-slate-500">Avg Price</p>
          <p className="text-slate-300">{position.avg_price_cents}¢</p>
        </div>
        <div>
          <p className="text-slate-500">Cost</p>
          <p className="text-slate-300">${position.cost_basis_usd.toFixed(2)}</p>
        </div>
        <div>
          <p className="text-slate-500">Value</p>
          <p className="text-slate-300">${position.current_value_usd.toFixed(2)}</p>
        </div>
      </div>
    </div>
  );
}
