"use client";

import { useAuth } from "@/hooks/useAuth";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, Suspense } from "react";

function LoginContent() {
  const { ready, authenticated, login } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    if (ready && authenticated) {
      const next = searchParams.get("next") || "/";
      router.replace(next);
    }
  }, [ready, authenticated, router, searchParams]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-primary">
      <div className="relative z-10 text-center max-w-md mx-auto px-6">
        {/* Floating shield illustration */}
        <div className="float-gentle mb-8">
          <svg width="80" height="80" viewBox="0 0 64 64" fill="none" className="mx-auto">
            <circle cx="32" cy="32" r="28" fill="var(--teal-light)" stroke="var(--ink)" strokeWidth="3"/>
            <path d="M22 28 Q22 20, 32 20 Q42 20, 42 28 L42 34 Q42 42, 32 44 Q22 42, 22 34Z" fill="var(--teal)" stroke="var(--ink)" strokeWidth="2"/>
            <polyline points="26,32 30,36 38,26" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </div>

        <div className="mb-6">
          <h1 className="text-5xl font-display text-ink tracking-tight">
            Eigen<span className="text-coral">Inference</span>
          </h1>
          <p className="mt-2 text-sm text-text-tertiary font-display text-lg">
            An Eigen Labs Research Project
          </p>
        </div>

        <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-gold-light border-2 border-gold text-ink text-xs font-bold mb-6 font-display">
          Experimental Research Preview
        </div>

        <p className="text-text-secondary text-base mb-8 leading-relaxed">
          Private AI inference through hardware-attested Apple Silicon providers.
          Your prompts stay encrypted, your data stays yours.
        </p>

        <button
          onClick={login}
          disabled={!ready}
          className="inline-flex items-center justify-center gap-2 px-8 py-3 rounded-xl
                     bg-coral text-white font-bold text-base border-[3px] border-ink
                     hover:translate-x-[-2px] hover:translate-y-[-2px] hover:shadow-[4px_4px_0_var(--ink)]
                     disabled:opacity-40 disabled:cursor-not-allowed
                     transition-all focus-ring"
        >
          {!ready ? "Loading..." : "Sign In"}
        </button>

        <p className="mt-6 text-xs text-text-tertiary">
          Sign in with email, wallet, or social account
        </p>

        <p className="mt-8 text-xs text-text-tertiary leading-relaxed max-w-sm mx-auto">
          This is an experimental research project by Eigen Labs.
          The service is provided as-is for research and evaluation purposes.
          Not intended for production workloads.
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginContent />
    </Suspense>
  );
}
