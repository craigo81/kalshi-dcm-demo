// KYC Verification Form
// Core Principle 17: Fitness Standards - Identity verification

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Shield, Upload, CheckCircle, Clock, XCircle, Loader2 } from 'lucide-react';
import { kycAPI } from '../../api/client';
import { useAuth } from '../../context/AuthContext';

export function KYCForm() {
  const navigate = useNavigate();
  const { refreshProfile, kyc } = useAuth();

  const [status, setStatus] = useState<string>('not_started');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const [formData, setFormData] = useState({
    document_type: 'drivers_license',
    document_number: '',
  });

  useEffect(() => {
    checkStatus();
  }, []);

  const checkStatus = async () => {
    try {
      const result = await kycAPI.getStatus();
      if ('status' in result) {
        setStatus(result.status);
      }
    } catch {
      // No KYC record yet
    } finally {
      setLoading(false);
    }
  };

  // Poll for status changes (demo auto-approval)
  useEffect(() => {
    if (status === 'pending') {
      const interval = setInterval(async () => {
        await checkStatus();
        await refreshProfile();
      }, 2000);
      return () => clearInterval(interval);
    }
  }, [status, refreshProfile]);

  // Redirect when approved
  useEffect(() => {
    if (status === 'approved' || kyc?.status === 'approved') {
      setTimeout(() => navigate('/dashboard'), 1500);
    }
  }, [status, kyc, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);

    try {
      await kycAPI.submit(formData);
      setStatus('pending');
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
      </div>
    );
  }

  // Status display
  if (status === 'pending') {
    return (
      <div className="min-h-screen flex items-center justify-center px-4">
        <div className="max-w-md w-full text-center">
          <div className="w-16 h-16 bg-amber-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
            <Clock className="w-8 h-8 text-amber-400 animate-pulse" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Verification In Progress</h1>
          <p className="text-slate-400 mb-6">
            We're reviewing your documents. This typically takes 1-3 business days.
          </p>
          <div className="card">
            <div className="flex items-center gap-3 text-left">
              <div className="w-2 h-2 bg-amber-400 rounded-full animate-pulse" />
              <span className="text-slate-300">Document review in progress...</span>
            </div>
          </div>
          <p className="text-xs text-slate-500 mt-4">
            Demo mode: Auto-approving in a few seconds...
          </p>
        </div>
      </div>
    );
  }

  if (status === 'approved') {
    return (
      <div className="min-h-screen flex items-center justify-center px-4">
        <div className="max-w-md w-full text-center">
          <div className="w-16 h-16 bg-emerald-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
            <CheckCircle className="w-8 h-8 text-emerald-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Verification Complete!</h1>
          <p className="text-slate-400 mb-6">
            Your identity has been verified. You can now start trading.
          </p>
          <button
            onClick={() => navigate('/dashboard')}
            className="btn btn-primary"
          >
            Go to Dashboard
          </button>
        </div>
      </div>
    );
  }

  if (status === 'rejected') {
    return (
      <div className="min-h-screen flex items-center justify-center px-4">
        <div className="max-w-md w-full text-center">
          <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
            <XCircle className="w-8 h-8 text-red-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Verification Failed</h1>
          <p className="text-slate-400 mb-6">
            We couldn't verify your identity. Please try again with valid documents.
          </p>
          <button
            onClick={() => setStatus('not_started')}
            className="btn btn-primary"
          >
            Try Again
          </button>
        </div>
      </div>
    );
  }

  // KYC Form
  return (
    <div className="min-h-screen flex items-center justify-center px-4 py-12">
      <div className="max-w-lg w-full">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-primary-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <Shield className="w-8 h-8 text-primary-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Identity Verification</h1>
          <p className="text-slate-400">
            Complete KYC to start trading. This is required by CFTC regulations.
          </p>
        </div>

        <div className="card">
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
              <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                {error}
              </div>
            )}

            {/* Document Type */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Document Type
              </label>
              <select
                value={formData.document_type}
                onChange={(e) => setFormData(prev => ({ ...prev, document_type: e.target.value }))}
                className="w-full"
              >
                <option value="drivers_license">Driver's License</option>
                <option value="passport">US Passport</option>
                <option value="state_id">State ID</option>
              </select>
            </div>

            {/* Document Number */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Document Number
              </label>
              <input
                type="text"
                value={formData.document_number}
                onChange={(e) => setFormData(prev => ({ ...prev, document_number: e.target.value }))}
                required
                placeholder="Enter document number"
                className="w-full"
              />
            </div>

            {/* Mock Upload Area */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Upload Document (Demo)
              </label>
              <div className="border-2 border-dashed border-slate-600 rounded-lg p-8 text-center hover:border-primary-500 transition-colors cursor-pointer">
                <Upload className="w-8 h-8 text-slate-500 mx-auto mb-2" />
                <p className="text-slate-400 text-sm">
                  Click to upload or drag and drop
                </p>
                <p className="text-slate-500 text-xs mt-1">
                  (Mock upload - no actual file required for demo)
                </p>
              </div>
            </div>

            {/* Consent */}
            <div className="p-4 bg-slate-700/30 rounded-lg">
              <p className="text-xs text-slate-400">
                By submitting, I consent to the verification of my identity information
                in accordance with the platform's Privacy Policy and CFTC requirements
                for participant eligibility (Core Principle 17).
              </p>
            </div>

            <button
              type="submit"
              disabled={submitting}
              className="btn btn-primary w-full flex items-center justify-center gap-2"
            >
              {submitting ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : (
                <Shield className="w-5 h-5" />
              )}
              Submit for Verification
            </button>
          </form>
        </div>

        <p className="text-xs text-slate-500 text-center mt-6">
          Your information is encrypted and stored securely. We comply with all
          applicable data protection regulations.
        </p>
      </div>
    </div>
  );
}
