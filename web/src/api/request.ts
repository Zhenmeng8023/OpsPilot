import type { ApiResponse } from "./types";

type RequestOptions = RequestInit & {
  skipAuth?: boolean;
};

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";
const TOKEN_KEY = "opspilot.accessToken";
const REFRESH_TOKEN_KEY = "opspilot.refreshToken";

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", headers.get("Content-Type") ?? "application/json");

  const token = localStorage.getItem(TOKEN_KEY);
  if (token && !options.skipAuth) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });

  if (response.status === 401) {
    localStorage.removeItem(TOKEN_KEY);
  }

  const body = (await response.json()) as ApiResponse<T>;
  if (!response.ok || body.code !== 0) {
    throw new Error(body.message || `Request failed with status ${response.status}`);
  }

  return body.data;
}

export { REFRESH_TOKEN_KEY, TOKEN_KEY };
