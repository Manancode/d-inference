"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/hooks/useAuth";
import { Loader2, DollarSign, Briefcase, TrendingUp, LogIn } from "lucide-react";

interface Earning {
  id: number;
  provider_id: string;
  provider_key: string;
  job_id: string;
  model: string;
  amount_micro_usd: number;
  prompt_tokens: number;
  completion_tokens: number;
  created_at: string;
}

interface EarningsResponse {
  account_id: string;
  earnings: Earning[];
  total_micro_usd: number;
  total_usd: string;
  count: number;
}

export default function EarningsPage() {
  const { authenticated, login } = useAuth();
  const [data, setData] = useState<EarningsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authenticated) {
      setLoading(false);
      return;
    }

    async function fetchEarnings() {
      try {
        // Use the API key stored by useAuth for authentication
        const apiKey = localStorage.getItem("eigeninference_api_key") || "";
        const coordinatorUrl =
          localStorage.getItem("eigeninference_coordinator_url") ||
          process.env.NEXT_PUBLIC_COORDINATOR_URL ||
          "https://inference-test.openinnovation.dev";

        const res = await fetch(
          `${coordinatorUrl}/v1/provider/account-earnings?limit=100`,
          {
            headers: {
              ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
            },
          }
        );
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        setData(await res.json());
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    }
    fetchEarnings();
    const interval = setInterval(fetchEarnings, 30000);
    return () => clearInterval(interval);
  }, [authenticated]);

  if (!authenticated) {
    return (
      <div className="max-w-4xl mx-auto p-6">
        <div className="text-center py-16">
          <LogIn size={32} className="mx-auto mb-3 text-text-tertiary opacity-50" />
          <p className="text-sm text-text-tertiary mb-4">
            Sign in to view your provider earnings.
          </p>
          <button
            onClick={login}
            className="px-4 py-2 rounded-xl bg-accent-brand text-white text-sm font-medium hover:bg-accent-brand-hover transition-colors"
          >
            Sign In
          </button>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 size={24} className="animate-spin text-accent-brand" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-4xl mx-auto p-6">
        <p className="text-accent-red text-sm">Failed to load earnings: {error}</p>
      </div>
    );
  }

  const totalEarned = data?.total_usd || "0.000000";
  const jobCount = data?.count || 0;

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-text-primary">Provider Earnings</h2>
        <p className="text-sm text-text-tertiary mt-0.5">
          Across all linked provider nodes
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-xl bg-bg-secondary shadow-sm p-5">
          <div className="flex items-center gap-2 mb-2">
            <DollarSign size={16} className="text-accent-green" />
            <p className="text-xs text-text-tertiary">Total Earned</p>
          </div>
          <p className="text-2xl font-bold text-text-primary">
            ${totalEarned}
          </p>
        </div>
        <div className="rounded-xl bg-bg-secondary shadow-sm p-5">
          <div className="flex items-center gap-2 mb-2">
            <Briefcase size={16} className="text-accent-amber" />
            <p className="text-xs text-text-tertiary">Jobs Completed</p>
          </div>
          <p className="text-2xl font-bold text-text-primary">
            {jobCount}
          </p>
        </div>
        <div className="rounded-xl bg-bg-secondary shadow-sm p-5">
          <div className="flex items-center gap-2 mb-2">
            <TrendingUp size={16} className="text-accent-brand" />
            <p className="text-xs text-text-tertiary">Avg per Job</p>
          </div>
          <p className="text-2xl font-bold text-text-primary">
            ${jobCount > 0 ? (parseFloat(totalEarned) / jobCount).toFixed(6) : "0.00"}
          </p>
        </div>
      </div>

      {/* Earnings history */}
      <div>
        <h3 className="text-sm font-semibold text-text-primary mb-3">Recent Activity</h3>
        <div className="rounded-xl bg-bg-secondary shadow-sm overflow-hidden">
          {data?.earnings && data.earnings.length > 0 ? (
            <table className="w-full">
              <thead>
                <tr className="border-b border-border-dim">
                  <th className="text-left text-xs text-text-tertiary font-medium px-4 py-3">Model</th>
                  <th className="text-left text-xs text-text-tertiary font-medium px-4 py-3">Earned</th>
                  <th className="text-left text-xs text-text-tertiary font-medium px-4 py-3">Tokens</th>
                  <th className="text-left text-xs text-text-tertiary font-medium px-4 py-3">Time</th>
                </tr>
              </thead>
              <tbody>
                {data.earnings.map((e) => (
                  <tr key={e.id} className="border-b border-border-dim/50 last:border-0">
                    <td className="px-4 py-3 text-sm font-mono text-text-primary">
                      {e.model.split("/").pop()}
                    </td>
                    <td className="px-4 py-3 text-sm font-mono text-accent-green">
                      +${(e.amount_micro_usd / 1_000_000).toFixed(6)}
                    </td>
                    <td className="px-4 py-3 text-sm text-text-tertiary">
                      {e.prompt_tokens + e.completion_tokens} ({e.completion_tokens} out)
                    </td>
                    <td className="px-4 py-3 text-sm text-text-tertiary">
                      {new Date(e.created_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="text-center py-12 text-text-tertiary">
              <p className="text-sm">No earnings activity yet</p>
              <p className="text-xs mt-1">Earnings appear here when your provider serves inference requests</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
