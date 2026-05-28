import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  if (!code || !state) {
    return NextResponse.redirect(new URL("/system/integrations?error=missing_params", request.url));
  }

  try {
    const res = await fetch(`http://localhost:8080/api/oauth/facebook/callback?code=${code}&state=${state}`);
    
    if (res.ok) {
      return NextResponse.redirect(new URL("/system/integrations?success=facebook_connected", request.url));
    } else {
      return NextResponse.redirect(new URL("/system/integrations?error=backend_failed", request.url));
    }
  } catch (err) {
    return NextResponse.redirect(new URL("/system/integrations?error=server_error", request.url));
  }
}
