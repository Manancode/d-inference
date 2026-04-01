import { NextRequest, NextResponse } from "next/server";

const IS_PRIVY_CONFIGURED =
  process.env.NEXT_PUBLIC_PRIVY_APP_ID &&
  process.env.NEXT_PUBLIC_PRIVY_APP_ID !== "placeholder";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Always allow through: login page, device linking, API routes, static assets
  if (
    pathname === "/login" ||
    pathname === "/link" ||
    pathname.startsWith("/api/") ||
    pathname.startsWith("/_next/") ||
    pathname.startsWith("/favicon")
  ) {
    return NextResponse.next();
  }

  // Skip auth when Privy is not configured (dev/placeholder mode)
  if (!IS_PRIVY_CONFIGURED) {
    return NextResponse.next();
  }

  // Check for Privy auth token cookie
  const privyToken = request.cookies.get("privy-token");
  if (privyToken?.value) {
    return NextResponse.next();
  }

  // No auth — redirect to login
  const loginUrl = request.nextUrl.clone();
  loginUrl.pathname = "/login";
  loginUrl.searchParams.set("next", pathname);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
