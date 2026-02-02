import type { Market } from '../types';

interface MarketsPanelProps {
  markets: Market[];
  onHalt: (ticker: string, reason: string) => void;
  onResume: (ticker: string) => void;
}

export function MarketsPanel({ markets, onHalt, onResume }: MarketsPanelProps) {
  const handleHalt = (ticker: string) => {
    const reason = window.prompt('Halt reason:');
    if (reason) {
      onHalt(ticker, reason);
    }
  };

  return (
    <div className="bg-dark-800 rounded-lg border border-dark-700">
      <div className="px-4 py-3 border-b border-dark-700">
        <h2 className="font-semibold">Market Status</h2>
      </div>
      <div className="p-4 space-y-3 max-h-96 overflow-y-auto">
        {markets.length === 0 ? (
          <p className="text-gray-500 text-center py-4">No markets available</p>
        ) : (
          markets.map((market) => (
            <div
              key={market.ticker}
              className="bg-dark-700 rounded p-3 flex items-center justify-between"
            >
              <div>
                <div className="flex items-center space-x-2">
                  <span className="font-medium">{market.ticker}</span>
                  {market.is_halted ? (
                    <span className="px-2 py-0.5 text-xs bg-red-600 rounded">
                      HALTED
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 text-xs bg-green-600 rounded">
                      OPEN
                    </span>
                  )}
                </div>
                <div className="text-sm text-gray-400 mt-1">
                  Last: {market.last_price}¢ | 24h Vol:{' '}
                  {market.volume_24h.toLocaleString()}
                </div>
                {market.halt_reason && (
                  <div className="text-xs text-red-400 mt-1">
                    {market.halt_reason}
                  </div>
                )}
              </div>
              <div className="flex space-x-2">
                {market.is_halted ? (
                  <button
                    onClick={() => onResume(market.ticker)}
                    className="px-3 py-1 text-xs bg-green-600 hover:bg-green-700 rounded transition"
                  >
                    Resume
                  </button>
                ) : (
                  <button
                    onClick={() => handleHalt(market.ticker)}
                    className="px-3 py-1 text-xs bg-red-600 hover:bg-red-700 rounded transition"
                  >
                    Halt
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
