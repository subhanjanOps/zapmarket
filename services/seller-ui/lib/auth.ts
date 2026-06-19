// Token is stored in an httpOnly cookie managed by the BFF routes in /app/api/auth/.
// JavaScript code never touches the raw JWT.

export async function logout(): Promise<void> {
  await fetch("/api/auth/logout", { method: "POST", cache: "no-store" });
}
