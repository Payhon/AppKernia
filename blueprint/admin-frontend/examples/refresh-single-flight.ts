let refreshPromise: Promise<string> | null = null;
async function requestNewAccessToken(): Promise<string> {
  const response = await fetch('/admin-api/v1/auth/token/refresh', { method: 'POST', credentials: 'include', headers: { 'X-CSRF-Token': readCsrfToken() } });
  if (!response.ok) throw new Error('Refresh failed');
  const body = (await response.json()) as { data: { accessToken: string } };
  return body.data.accessToken;
}
export function refreshAccessTokenSingleFlight(): Promise<string> {
  if (!refreshPromise) refreshPromise = requestNewAccessToken().finally(() => { refreshPromise = null; });
  return refreshPromise;
}
declare function readCsrfToken(): string;
