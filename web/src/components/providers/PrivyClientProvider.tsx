"use client";

import { createContext, useContext } from "react";
import { PrivyProvider, usePrivy } from "@privy-io/react-auth";

const PRIVY_APP_ID = process.env.NEXT_PUBLIC_PRIVY_APP_ID || "";
const IS_PRIVY_CONFIGURED = PRIVY_APP_ID && PRIVY_APP_ID !== "placeholder";

export interface AuthState {
  ready: boolean;
  authenticated: boolean;
  user: unknown;
  login: () => void;
  logout: () => Promise<void>;
}

const MOCK_AUTH: AuthState = {
  ready: true,
  authenticated: true,
  user: null,
  login: () => {},
  logout: async () => {},
};

const AuthContext = createContext<AuthState>(MOCK_AUTH);

/** Provides the raw Privy/mock auth state to the tree. */
export function useAuthContext() {
  return useContext(AuthContext);
}

/** Bridges usePrivy() into our AuthContext when Privy is active. */
function PrivyAuthBridge({ children }: { children: React.ReactNode }) {
  const privy = usePrivy();
  return <AuthContext.Provider value={privy}>{children}</AuthContext.Provider>;
}

export function PrivyClientProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  // When Privy is not configured, provide mock auth without PrivyProvider
  if (!IS_PRIVY_CONFIGURED) {
    return (
      <AuthContext.Provider value={MOCK_AUTH}>
        {children}
      </AuthContext.Provider>
    );
  }

  return (
    <PrivyProvider
      appId={PRIVY_APP_ID}
      config={{
        loginMethods: ["email", "wallet", "google", "github"],
        appearance: {
          theme: "light",
          accentColor: "#6366f1",
          logo: undefined,
        },
        embeddedWallets: {
          solana: {
            createOnLogin: "all-users",
          },
        },
      }}
    >
      <PrivyAuthBridge>{children}</PrivyAuthBridge>
    </PrivyProvider>
  );
}
