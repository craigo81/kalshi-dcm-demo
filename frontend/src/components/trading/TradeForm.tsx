// Trade Form Component
// Core Principle 9: Fair and equitable execution
// Core Principle 11: Pre-trade margin checks (100% collateralization)
// Core Principle 5: Position limits enforcement

import React, { useState, useEffect, useMemo } from 'react';
import { X, TrendingUp, TrendingDown, AlertTriangle, CheckCircle, Loader2, Shield, Clock, Building2, ShieldCheck } from 'lucide-react';
import { Market, tradingAPI, PreTradeCheck } from '../../api/client';
import { useAuth } from '../../context/AuthContext';

// Risk badge colors based on market risk category (CP 3)
const riskColors: Record<string, string> = {
  low: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
  medium: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  high: 'bg-red-500/20 text-red-400 border-red-500/30',
};

// Format time remaining until market close
function formatTimeRemaining(closeTime: string): string {
  const close = new Date(closeTime);
  const now = new Date();
  const diff = close.getTime() - now.getTime();

  if (diff <= 0) return 'Closed';

  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

interface TradeFormProps {
  market: Market;
  onClose: () => void;
  onSuccess: () => void;
}

export function TradeForm({ market, onClose, onSuccess }: TradeFormProps) {
  const { wallet, isVerified, refreshProfile } = useAuth();

  const [side, setSide] = useState<'yes' | 'no'>('yes');
  const [quantity, setQuantity] = useState(10);
  const [priceCents, setPriceCents] = useState(market.yes_bid || 50);
  const [preCheck, setPreCheck] = useState<PreTradeCheck | null>(null);
  const [loading, setLoading] = useState(false);
  const [checkLoading, setCheckLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  // Calculate collateral required (Core Principle 11: 100% margin)
  const collateralRequired = side === 'yes'
    ? (quantity * priceCents) / 100
    : (quantity * (100 - priceCents)) / 100;

  // Max payout if contract settles in your favor
  const maxPayout = quantity;
  const maxProfit = maxPayout - collateralRequired;

  // Run pre-trade check when inputs change
  useEffect(() => {
    const runCheck = async () => {
      if (!isVerified) return;

      setCheckLoading(true);
      try {
        const check = await tradingAPI.preCheck({
          market_ticker: market.ticker,
          side,
          quantity,
          price_cents: priceCents,
        });
        setPreCheck(check);
      } catch {
        // Silently fail - user will see error on submit
      } finally {
        setCheckLoading(false);
      }
    };

    const debounce = setTimeout(runCheck, 300);
    return () => clearTimeout(debounce);
  }, [market.ticker, side, quantity, priceCents, isVerified]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await tradingAPI.placeOrder({
        market_ticker: market.ticker,
        side,
        type: 'limit',
        quantity,
        price_cents: priceCents,
      });

      setSuccess(true);
      await refreshProfile();

      setTimeout(() => {
        onSuccess();
        onClose();
      }, 1500);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Order failed');
    } finally {
      setLoading(false);
    }
  };

  if (!isVerified) {
    return (
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
        <div className="bg-slate-900 rounded-2xl w-full max-w-md p-6 border border-slate-700">
          <div className="text-center">
            <Shield className="w-12 h-12 text-amber-400 mx-auto mb-4" />
            <h2 className="text-xl font-bold text-white mb-2">KYC Required</h2>
            <p className="text-slate-400 mb-6">
              You must complete identity verification before trading.
              This is required by CFTC Core Principle 17.
            </p>
            <button onClick={onClose} className="btn btn-primary">
              Complete KYC
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (success) {
    return (
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
        <div className="bg-slate-900 rounded-2xl w-full max-w-md p-6 border border-slate-700 text-center">
          <CheckCircle className="w-16 h-16 text-emerald-400 mx-auto mb-4" />
          <h2 className="text-xl font-bold text-white mb-2">Order Placed!</h2>
          <p className="text-slate-400">
            Your {side.toUpperCase()} order for {quantity} contracts has been submitted.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-slate-900 rounded-2xl w-full max-w-lg border border-slate-700 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-slate-900 border-b border-slate-700 p-4 flex justify-between items-start">
          <div>
            <h2 className="text-lg font-bold text-white">{market.title}</h2>
            <p className="text-sm text-slate-400">{market.ticker}</p>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-slate-800 rounded-lg">
            <X className="w-5 h-5 text-slate-400" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {/* Market Info Banner */}
          <div className="flex flex-wrap items-center gap-2 p-3 bg-slate-800/50 rounded-lg">
            {/* Risk Category (CP 3) */}
            <div className={`flex items-center gap-1.5 px-2 py-1 rounded border text-xs ${riskColors[market.risk_category || 'low']}`}>
              <ShieldCheck className="w-3 h-3" />
              <span className="font-medium">{(market.risk_category || 'low').toUpperCase()} RISK</span>
            </div>

            {/* Exchange Routing */}
            <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30 text-xs">
              <Building2 className="w-3 h-3" />
              <span className="font-medium">Kalshi DCM</span>
            </div>

            {/* Time to Close */}
            {market.close_time && (
              <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-slate-700 text-slate-300 text-xs ml-auto">
                <Clock className="w-3 h-3" />
                <span>{formatTimeRemaining(market.close_time)}</span>
              </div>
            )}
          </div>

          {/* Side Selection */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-3">
              Position
            </label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setSide('yes')}
                className={`p-4 rounded-lg border-2 transition-all ${
                  side === 'yes'
                    ? 'border-emerald-500 bg-emerald-500/10'
                    : 'border-slate-600 hover:border-slate-500'
                }`}
              >
                <div className="flex items-center gap-2 mb-2">
                  <TrendingUp className={`w-5 h-5 ${side === 'yes' ? 'text-emerald-400' : 'text-slate-400'}`} />
                  <span className={`font-semibold ${side === 'yes' ? 'text-emerald-400' : 'text-slate-300'}`}>
                    YES
                  </span>
                </div>
                <p className="text-2xl font-bold text-emerald-400">{market.yes_bid}¢</p>
              </button>

              <button
                type="button"
                onClick={() => setSide('no')}
                className={`p-4 rounded-lg border-2 transition-all ${
                  side === 'no'
                    ? 'border-red-500 bg-red-500/10'
                    : 'border-slate-600 hover:border-slate-500'
                }`}
              >
                <div className="flex items-center gap-2 mb-2">
                  <TrendingDown className={`w-5 h-5 ${side === 'no' ? 'text-red-400' : 'text-slate-400'}`} />
                  <span className={`font-semibold ${side === 'no' ? 'text-red-400' : 'text-slate-300'}`}>
                    NO
                  </span>
                </div>
                <p className="text-2xl font-bold text-red-400">{market.no_bid || 100 - market.yes_bid}¢</p>
              </button>
            </div>
          </div>

          {/* Quantity */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Contracts
            </label>
            <input
              type="number"
              value={quantity}
              onChange={(e) => setQuantity(Math.max(1, Math.min(1000, parseInt(e.target.value) || 1)))}
              min="1"
              max="1000"
              className="w-full"
            />
            <div className="flex gap-2 mt-2">
              {[10, 50, 100, 500].map((q) => (
                <button
                  key={q}
                  type="button"
                  onClick={() => setQuantity(q)}
                  className="flex-1 py-1 px-2 bg-slate-700 hover:bg-slate-600 rounded text-sm text-white"
                >
                  {q}
                </button>
              ))}
            </div>
          </div>

          {/* Price */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Limit Price (¢)
            </label>
            <input
              type="number"
              value={priceCents}
              onChange={(e) => setPriceCents(Math.max(1, Math.min(99, parseInt(e.target.value) || 50)))}
              min="1"
              max="99"
              className="w-full"
            />
            <input
              type="range"
              value={priceCents}
              onChange={(e) => setPriceCents(parseInt(e.target.value))}
              min="1"
              max="99"
              className="w-full mt-2"
            />
          </div>

          {/* Order Summary */}
          <div className="p-4 bg-slate-800/50 rounded-lg space-y-3">
            <h4 className="font-medium text-white">Order Summary</h4>

            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Collateral Required</span>
              <span className="text-white font-medium">${collateralRequired.toFixed(2)}</span>
            </div>

            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Max Payout</span>
              <span className="text-emerald-400">${maxPayout.toFixed(2)}</span>
            </div>

            <div className="flex justify-between text-sm">
              <span className="text-slate-400">Max Profit</span>
              <span className="text-emerald-400">+${maxProfit.toFixed(2)}</span>
            </div>

            <div className="flex justify-between text-sm pt-2 border-t border-slate-700">
              <span className="text-slate-400">Available Balance</span>
              <span className="text-white">${wallet?.available_usd.toFixed(2) || '0.00'}</span>
            </div>
          </div>

          {/* Pre-trade Check Results */}
          {checkLoading && (
            <div className="flex items-center gap-2 text-slate-400 text-sm">
              <Loader2 className="w-4 h-4 animate-spin" />
              Checking order...
            </div>
          )}

          {preCheck && !checkLoading && (
            <div className={`p-4 rounded-lg ${preCheck.passed ? 'bg-emerald-500/10 border border-emerald-500/20' : 'bg-red-500/10 border border-red-500/20'}`}>
              <div className="flex items-center gap-2 mb-2">
                {preCheck.passed ? (
                  <CheckCircle className="w-5 h-5 text-emerald-400" />
                ) : (
                  <AlertTriangle className="w-5 h-5 text-red-400" />
                )}
                <span className={`font-medium ${preCheck.passed ? 'text-emerald-400' : 'text-red-400'}`}>
                  {preCheck.passed ? 'Pre-trade Check Passed' : 'Pre-trade Check Failed'}
                </span>
              </div>

              {preCheck.errors?.length > 0 && (
                <ul className="list-disc list-inside text-sm text-red-400 space-y-1">
                  {preCheck.errors.map((err, i) => (
                    <li key={i}>{err}</li>
                  ))}
                </ul>
              )}

              {preCheck.warnings?.length > 0 && (
                <ul className="list-disc list-inside text-sm text-amber-400 space-y-1 mt-2">
                  {preCheck.warnings.map((warn, i) => (
                    <li key={i}>{warn}</li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {/* Error */}
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
              {error}
            </div>
          )}

          {/* Submit */}
          <button
            type="submit"
            disabled={loading || (preCheck && !preCheck.passed)}
            className={`w-full py-3 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 ${
              side === 'yes'
                ? 'bg-emerald-600 hover:bg-emerald-700 text-white'
                : 'bg-red-600 hover:bg-red-700 text-white'
            } disabled:opacity-50 disabled:cursor-not-allowed`}
          >
            {loading ? (
              <Loader2 className="w-5 h-5 animate-spin" />
            ) : (
              <>
                {side === 'yes' ? <TrendingUp className="w-5 h-5" /> : <TrendingDown className="w-5 h-5" />}
                Place {side.toUpperCase()} Order
              </>
            )}
          </button>

          {/* Compliance Notice */}
          <p className="text-xs text-slate-500 text-center">
            Orders are subject to CFTC Core Principles including 100% collateralization
            (CP 11) and position limits (CP 5). Orders route to Kalshi DCM.
          </p>
        </form>
      </div>
    </div>
  );
}
