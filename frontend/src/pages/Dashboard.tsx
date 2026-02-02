// Dashboard Page
// Main trading interface with markets, wallet, and positions

import React, { useState, useEffect } from 'react';
import { TrendingUp, Activity, AlertTriangle, RefreshCw, Loader2, Filter } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { marketsAPI, Market } from '../api/client';
import { WalletPanel } from '../components/wallet/WalletPanel';
import { PositionsPanel } from '../components/portfolio/PositionsPanel';
import { MarketCard } from '../components/trading/MarketCard';
import { TradeForm } from '../components/trading/TradeForm';

export function Dashboard() {
  const { user, wallet, isVerified, refreshProfile } = useAuth();

  const [markets, setMarkets] = useState<Market[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('open');

  const fetchMarkets = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const response = await marketsAPI.list({ status: statusFilter, limit: 20 });
      setMarkets(response.data || []);
    } catch (error) {
      console.error('Failed to fetch markets:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    fetchMarkets();
  }, [statusFilter]);

  const handleTradeSuccess = () => {
    refreshProfile();
    fetchMarkets(true);
  };

  // Calculate exposure utilization
  const exposure = wallet?.locked_usd || 0;
  const limit = user?.position_limit_usd || 25000;
  const utilization = (exposure / limit) * 100;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* Welcome Banner */}
        {!isVerified && (
          <div className="mb-6 p-4 bg-amber-500/10 border border-amber-500/20 rounded-xl flex items-center gap-4">
            <AlertTriangle className="w-6 h-6 text-amber-400 flex-shrink-0" />
            <div>
              <p className="text-amber-200 font-medium">Complete Identity Verification</p>
              <p className="text-amber-200/70 text-sm">
                You must complete KYC verification before you can trade. This is required by CFTC Core Principle 17.
              </p>
            </div>
            <a href="/kyc" className="btn btn-primary ml-auto">
              Verify Now
            </a>
          </div>
        )}

        {/* Stats Row */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="card">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-emerald-500/20 rounded-lg flex items-center justify-center">
                <TrendingUp className="w-5 h-5 text-emerald-400" />
              </div>
              <div>
                <p className="text-sm text-slate-400">Available</p>
                <p className="text-xl font-bold text-white">${wallet?.available_usd.toFixed(2) || '0.00'}</p>
              </div>
            </div>
          </div>

          <div className="card">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-amber-500/20 rounded-lg flex items-center justify-center">
                <Activity className="w-5 h-5 text-amber-400" />
              </div>
              <div>
                <p className="text-sm text-slate-400">Locked</p>
                <p className="text-xl font-bold text-white">${wallet?.locked_usd.toFixed(2) || '0.00'}</p>
              </div>
            </div>
          </div>

          <div className="card col-span-2">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-slate-400">Position Limit Utilization (Core Principle 5)</p>
              <p className="text-sm text-slate-300">{utilization.toFixed(1)}%</p>
            </div>
            <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${
                  utilization > 80 ? 'bg-red-500' : utilization > 50 ? 'bg-amber-500' : 'bg-emerald-500'
                }`}
                style={{ width: `${Math.min(utilization, 100)}%` }}
              />
            </div>
            <p className="text-xs text-slate-500 mt-1">
              ${exposure.toFixed(2)} / ${limit.toFixed(2)}
            </p>
          </div>
        </div>

        {/* Main Content */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Markets Column */}
          <div className="lg:col-span-2">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-white">Markets</h2>
              <div className="flex items-center gap-3">
                {/* Status Filter */}
                <div className="flex items-center gap-1 bg-slate-800 rounded-lg p-1">
                  <Filter className="w-4 h-4 text-slate-400 ml-2" />
                  {['open', 'closed', 'settled'].map((status) => (
                    <button
                      key={status}
                      onClick={() => setStatusFilter(status)}
                      className={`px-3 py-1 rounded text-sm transition-colors ${
                        statusFilter === status
                          ? 'bg-primary-600 text-white'
                          : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      {status.charAt(0).toUpperCase() + status.slice(1)}
                    </button>
                  ))}
                </div>
                <button
                  onClick={() => fetchMarkets(true)}
                  disabled={refreshing}
                  className="p-2 hover:bg-slate-800 rounded-lg transition-colors"
                >
                  <RefreshCw className={`w-4 h-4 text-slate-400 ${refreshing ? 'animate-spin' : ''}`} />
                </button>
              </div>
            </div>

            {loading ? (
              <div className="flex justify-center py-12">
                <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
              </div>
            ) : markets.length === 0 ? (
              <div className="text-center py-12">
                <TrendingUp className="w-12 h-12 text-slate-600 mx-auto mb-3" />
                <p className="text-slate-400">No markets found</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {markets.map((market) => (
                  <MarketCard
                    key={market.ticker}
                    market={market}
                    onSelect={setSelectedMarket}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            <WalletPanel wallet={wallet} onUpdate={refreshProfile} />
            <PositionsPanel />
          </div>
        </div>
      </div>

      {/* Trade Modal */}
      {selectedMarket && (
        <TradeForm
          market={selectedMarket}
          onClose={() => setSelectedMarket(null)}
          onSuccess={handleTradeSuccess}
        />
      )}
    </div>
  );
}
