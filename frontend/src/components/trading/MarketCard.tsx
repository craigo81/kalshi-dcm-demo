// Market Card Component
// Core Principle 3: Displays market info with risk classification

import React from 'react';
import { TrendingUp, TrendingDown, Clock, Activity, Shield } from 'lucide-react';
import { Market } from '../../api/client';

interface MarketCardProps {
  market: Market;
  onSelect: (market: Market) => void;
}

export function MarketCard({ market, onSelect }: MarketCardProps) {
  const yesPrice = market.yes_bid || market.last_price || 50;
  const noPrice = 100 - yesPrice;
  const isPriceUp = yesPrice >= 50;

  const formatPrice = (price: number) => `${price}¢`;

  const formatVolume = (volume: number) => {
    if (volume >= 1000000) return `${(volume / 1000000).toFixed(1)}M`;
    if (volume >= 1000) return `${(volume / 1000).toFixed(1)}K`;
    return volume.toString();
  };

  const getTimeRemaining = () => {
    if (!market.close_time) return null;
    const closeDate = new Date(market.close_time);
    const now = new Date();
    const diff = closeDate.getTime() - now.getTime();

    if (diff <= 0) return 'Closed';

    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h`;
    return '<1h';
  };

  const getRiskBadge = () => {
    switch (market.risk_category) {
      case 'low':
        return <span className="badge badge-success">Low Risk</span>;
      case 'medium':
        return <span className="badge badge-warning">Medium Risk</span>;
      case 'high':
        return <span className="badge badge-danger">Higher Risk</span>;
      default:
        return null;
    }
  };

  return (
    <div
      onClick={() => onSelect(market)}
      className="card hover:border-primary-500 transition-all duration-200 cursor-pointer group"
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1 pr-2">
          <h3 className="text-white font-semibold leading-tight group-hover:text-primary-400 transition-colors line-clamp-2">
            {market.title}
          </h3>
          {market.subtitle && (
            <p className="text-slate-400 text-sm mt-1 line-clamp-1">{market.subtitle}</p>
          )}
        </div>
        <div className="flex flex-col items-end gap-1">
          <span className={`badge ${market.status === 'open' ? 'badge-success' : 'badge-info'}`}>
            {market.status.toUpperCase()}
          </span>
          {getRiskBadge()}
        </div>
      </div>

      {/* Price Display */}
      <div className="grid grid-cols-2 gap-3 mb-4">
        <div className={`rounded-lg p-3 ${isPriceUp ? 'bg-emerald-500/10 border border-emerald-500/20' : 'bg-slate-700/30'}`}>
          <div className="flex items-center justify-between">
            <span className="text-slate-400 text-sm">YES</span>
            {isPriceUp && <TrendingUp className="w-4 h-4 text-emerald-400" />}
          </div>
          <div className={`text-2xl font-bold ${isPriceUp ? 'text-emerald-400' : 'text-slate-300'}`}>
            {formatPrice(yesPrice)}
          </div>
          <div className="text-xs text-slate-500">
            {yesPrice}% implied
          </div>
        </div>

        <div className={`rounded-lg p-3 ${!isPriceUp ? 'bg-red-500/10 border border-red-500/20' : 'bg-slate-700/30'}`}>
          <div className="flex items-center justify-between">
            <span className="text-slate-400 text-sm">NO</span>
            {!isPriceUp && <TrendingDown className="w-4 h-4 text-red-400" />}
          </div>
          <div className={`text-2xl font-bold ${!isPriceUp ? 'text-red-400' : 'text-slate-300'}`}>
            {formatPrice(noPrice)}
          </div>
          <div className="text-xs text-slate-500">
            {noPrice}% implied
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="flex items-center justify-between text-sm text-slate-400 border-t border-slate-700 pt-3">
        <div className="flex items-center gap-1">
          <Activity className="w-4 h-4" />
          <span>Vol: {formatVolume(market.volume_24h || market.volume)}</span>
        </div>
        <div className="flex items-center gap-1">
          <Clock className="w-4 h-4" />
          <span>{getTimeRemaining()}</span>
        </div>
      </div>

      {/* Category */}
      <div className="mt-3 flex items-center gap-2">
        <span className="badge badge-info">
          {market.category || market.series_ticker}
        </span>
        {market.risk_category === 'low' && (
          <div className="flex items-center gap-1 text-xs text-emerald-400">
            <Shield className="w-3 h-3" />
            <span>Economic Binary</span>
          </div>
        )}
      </div>
    </div>
  );
}
