interface EmergencyControlsProps {
  onGlobalHalt: () => void;
  onGlobalResume: () => void;
  isHalted: boolean;
}

export function EmergencyControls({
  onGlobalHalt,
  onGlobalResume,
  isHalted,
}: EmergencyControlsProps) {
  return (
    <div className="bg-red-900/20 border border-red-700 rounded-lg p-4 mb-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-red-400">Emergency Controls</h2>
          <p className="text-sm text-gray-400">
            Core Principle 4: Prevention of Market Disruption
          </p>
        </div>
        <div className="flex space-x-4">
          <button
            onClick={onGlobalHalt}
            disabled={isHalted}
            className={`px-4 py-2 rounded font-medium transition ${
              isHalted
                ? 'bg-gray-600 cursor-not-allowed'
                : 'bg-red-600 hover:bg-red-700'
            }`}
          >
            🛑 Global Halt
          </button>
          <button
            onClick={onGlobalResume}
            disabled={!isHalted}
            className={`px-4 py-2 rounded font-medium transition ${
              !isHalted
                ? 'bg-gray-600 cursor-not-allowed'
                : 'bg-green-600 hover:bg-green-700'
            }`}
          >
            ▶️ Resume All
          </button>
        </div>
      </div>
    </div>
  );
}
