import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const code = searchParams.get("code");
  const state = searchParams.get("state");

  if (!code || !state) {
    return NextResponse.redirect(new URL("/system/integrations?error=missing_params", request.url));
  }

  // Forward the code and state to the Go backend
  // We can just redirect the user's browser to the Go backend's GET callback!
  // Wait, the redirect URI must match exactly what Google authorized.
  // We can't redirect with auth code because Google already sent it to us.
  // Let's call the backend from here.
  
  try {
    // The backend's GET endpoint expects code and state.
    // Wait, if we call it from server-side, the backend sets the token in DB.
    const res = await fetch(`http://localhost:8080/api/oauth/google/callback?code=${code}&state=${state}`);
    
    if (res.ok) {
      return NextResponse.redirect(new URL("/system/integrations?success=google_connected", request.url));
    } else {
      return NextResponse.redirect(new URL("/system/integrations?error=backend_failed", request.url));
    }
  } catch (err) {
    return NextResponse.redirect(new URL("/system/integrations?error=server_error", request.url));
  }
}
