// Wallet Panel Component
// Core Principle 11: Financial Integrity - Shows available/locked funds
// Core Principle 13: Financial Resources - Segregated funds display

import React, { useState } from 'react';
import { Wallet, Plus, ArrowUpRight, ArrowDownRight, Loader2 } from 'lucide-react';
import { walletAPI, Wallet as WalletType } from '../../api/client';
import { useAuth } from '../../context/AuthContext';

interface WalletPanelProps {
  wallet: WalletType | null;
  onUpdate?: () => void;
}

export function WalletPanel({ wallet, onUpdate }: WalletPanelProps) {
  const { refreshProfile } = useAuth();
  const [showDeposit, setShowDeposit] = useState(false);
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const handleDeposit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);

    try {
      const depositAmount = parseFloat(amount);
      if (isNaN(depositAmount) || depositAmount <= 0) {
        throw new Error('Invalid amount');
      }

      await walletAPI.deposit(depositAmount);
      setSuccess(`Successfully deposited $${depositAmount.toFixed(2)}`);
      setAmount('');
      await refreshProfile();
      onUpdate?.();

      setTimeout(() => {
        setShowDeposit(false);
        setSuccess('');
      }, 2000);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } }; message?: string };
      setError(error.response?.data?.error || error.message || 'Deposit failed');
    } finally {
      setLoading(false);
    }
  };

  const quickAmounts = [100, 500, 1000, 5000];

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-primary-500/20 rounded-lg flex items-center justify-center">
            <Wallet className="w-5 h-5 text-primary-400" />
          </div>
          <div>
            <h3 className="font-semibold text-white">Wallet</h3>
            <p className="text-xs text-slate-400">Segregated Funds (Core Principle 13)</p>
          </div>
        </div>
        <button
          onClick={() => setShowDeposit(!showDeposit)}
          className="btn btn-primary btn-sm flex items-center gap-1"
        >
          <Plus className="w-4 h-4" />
          Deposit
        </button>
      </div>

      {/* Balance Display */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="p-4 bg-slate-700/30 rounded-lg">
          <p className="text-xs text-slate-400 mb-1">Available</p>
          <p className="text-xl font-bold text-emerald-400">
            ${wallet?.available_usd.toFixed(2) || '0.00'}
          </p>
        </div>
        <div className="p-4 bg-slate-700/30 rounded-lg">
          <p className="text-xs text-slate-400 mb-1">Locked</p>
          <p className="text-xl font-bold text-amber-400">
            ${wallet?.locked_usd.toFixed(2) || '0.00'}
          </p>
        </div>
        <div className="p-4 bg-slate-700/30 rounded-lg">
          <p className="text-xs text-slate-400 mb-1">Total</p>
          <p className="text-xl font-bold text-white">
            ${((wallet?.available_usd || 0) + (wallet?.locked_usd || 0)).toFixed(2)}
          </p>
        </div>
      </div>

      {/* Deposit Form */}
      {showDeposit && (
        <div className="border-t border-slate-700 pt-4">
          <form onSubmit={handleDeposit} className="space-y-4">
            {error && (
              <div className="p-2 bg-red-500/10 text-red-400 text-sm rounded">
                {error}
              </div>
            )}
            {success && (
              <div className="p-2 bg-emerald-500/10 text-emerald-400 text-sm rounded">
                {success}
              </div>
            )}

            <div>
              <label className="block text-sm text-slate-400 mb-2">
                Deposit Amount (USD)
              </label>
              <input
                type="number"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="0.00"
                min="1"
                max="10000"
                step="0.01"
                className="w-full"
              />
            </div>

            {/* Quick amounts */}
            <div className="flex gap-2">
              {quickAmounts.map((amt) => (
                <button
                  key={amt}
                  type="button"
                  onClick={() => setAmount(amt.toString())}
                  className="flex-1 py-2 px-3 bg-slate-700 hover:bg-slate-600 rounded text-sm text-white transition-colors"
                >
                  ${amt}
                </button>
              ))}
            </div>

            <button
              type="submit"
              disabled={loading || !amount}
              className="btn btn-success w-full flex items-center justify-center gap-2"
            >
              {loading ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <ArrowDownRight className="w-4 h-4" />
              )}
              Deposit Funds
            </button>

            <p className="text-xs text-slate-500 text-center">
              Demo mode: Funds are simulated. No real money involved.
            </p>
          </form>
        </div>
      )}

      {/* Recent Activity */}
      <div className="border-t border-slate-700 pt-4 mt-4">
        <h4 className="text-sm font-medium text-slate-300 mb-3">Recent Activity</h4>
        <div className="space-y-2">
          {wallet?.total_deposited ? (
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <ArrowDownRight className="w-4 h-4 text-emerald-400" />
                <span className="text-slate-300">Total Deposited</span>
              </div>
              <span className="text-emerald-400">+${wallet.total_deposited.toFixed(2)}</span>
            </div>
          ) : null}
          {wallet?.total_withdrawn ? (
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <ArrowUpRight className="w-4 h-4 text-red-400" />
                <span className="text-slate-300">Total Withdrawn</span>
              </div>
              <span className="text-red-400">-${wallet.total_withdrawn.toFixed(2)}</span>
            </div>
          ) : null}
          {!wallet?.total_deposited && !wallet?.total_withdrawn && (
            <p className="text-slate-500 text-sm">No transactions yet</p>
          )}
        </div>
      </div>
    </div>
  );
}
