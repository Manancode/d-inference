"use client";

import { useSearchParams } from "next/navigation";
import { useState, Suspense } from "react";

function GateForm() {
  const searchParams = useSearchParams();
  const next = searchParams.get("next") || "/";
  const [password, setPassword] = useState("");
  const [error, setError] = useState(false);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(false);
    // Submit via redirect so middleware can set the cookie
    window.location.href = `${next}?password=${encodeURIComponent(password)}`;
  }

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "#0a0a12",
        fontFamily:
          "var(--font-dm-sans), system-ui, -apple-system, sans-serif",
      }}
    >
      <div
        style={{
          width: 380,
          padding: "48px 36px",
          background: "#13131f",
          borderRadius: 16,
          border: "1px solid #2a2a3e",
          boxShadow: "0 20px 60px rgba(0,0,0,0.5)",
        }}
      >
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <div
            style={{
              fontSize: 28,
              fontWeight: 700,
              color: "#e0e0f0",
              letterSpacing: "-0.02em",
            }}
          >
            DGInf
          </div>
          <div
            style={{
              fontSize: 13,
              color: "#6b6b8a",
              marginTop: 6,
              letterSpacing: "0.04em",
              textTransform: "uppercase",
            }}
          >
            Private Inference Network
          </div>
        </div>

        <form onSubmit={handleSubmit}>
          <input
            type="password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              setError(false);
            }}
            placeholder="Access code"
            autoFocus
            style={{
              width: "100%",
              padding: "14px 16px",
              fontSize: 15,
              background: "#0a0a12",
              border: error ? "1px solid #dc2626" : "1px solid #2a2a3e",
              borderRadius: 10,
              color: "#e0e0f0",
              outline: "none",
              boxSizing: "border-box",
              transition: "border-color 0.15s",
            }}
            onFocus={(e) =>
              (e.target.style.borderColor = error ? "#dc2626" : "#7c3aed")
            }
            onBlur={(e) =>
              (e.target.style.borderColor = error ? "#dc2626" : "#2a2a3e")
            }
          />
          {error && (
            <div style={{ color: "#dc2626", fontSize: 13, marginTop: 8 }}>
              Incorrect access code
            </div>
          )}
          <button
            type="submit"
            style={{
              width: "100%",
              marginTop: 16,
              padding: "14px 0",
              fontSize: 15,
              fontWeight: 600,
              background: "#7c3aed",
              color: "#fff",
              border: "none",
              borderRadius: 10,
              cursor: "pointer",
              transition: "background 0.15s",
            }}
            onMouseOver={(e) =>
              (e.currentTarget.style.background = "#6d28d9")
            }
            onMouseOut={(e) =>
              (e.currentTarget.style.background = "#7c3aed")
            }
          >
            Enter
          </button>
        </form>
      </div>
    </div>
  );
}

export default function GatePage() {
  return (
    <Suspense>
      <GateForm />
    </Suspense>
  );
}
