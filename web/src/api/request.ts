import type { ApiResponse } from "./types";

type RequestOptions = RequestInit & {
  skipAuth?: boolean;
};

export class ApiError extends Error {
  status: number;
  code: number;
  traceId: string;
  debugMessage: string;

  constructor(args: { message: string; status?: number; code?: number; traceId?: string; debugMessage?: string }) {
    const message = formatMessage(args.message, args.traceId, args.debugMessage);
    super(message);
    this.name = "ApiError";
    this.status = args.status ?? 0;
    this.code = args.code ?? -1;
    this.traceId = args.traceId ?? "";
    this.debugMessage = args.debugMessage ?? "";
  }
}

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

  const body = await parseApiResponse<T>(response);
  if (!response.ok || !body || body.code !== 0) {
    throw new ApiError({
      message: body?.message || `Request failed with status ${response.status}`,
      status: response.status,
      code: body?.code,
      traceId: body?.traceId || response.headers.get("X-Trace-Id") || "",
      debugMessage: body?.debugMessage
    });
  }

  if (typeof body.data === "undefined") {
    throw new ApiError({
      message: "Response payload is missing data.",
      status: response.status,
      code: body.code,
      traceId: body.traceId || response.headers.get("X-Trace-Id") || "",
      debugMessage: body.debugMessage
    });
  }

  return body.data;
}

export async function streamSSE(
  path: string,
  handlers: {
    onEvent: (event: string, data: string) => void;
    onError?: (error: Error) => void;
    signal?: AbortSignal;
  }
) {
  const headers = new Headers();
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers,
    signal: handlers.signal
  });
  if (!response.ok || !response.body) {
    throw new Error(`Stream failed with status ${response.status}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    const chunks = buffer.split("\n\n");
    buffer = chunks.pop() ?? "";
    for (const chunk of chunks) {
      const event = chunk
        .split("\n")
        .find((line) => line.startsWith("event:"))
        ?.slice(6)
        .trim() ?? "message";
      const data = chunk
        .split("\n")
        .filter((line) => line.startsWith("data:"))
        .map((line) => line.slice(5).trimStart())
        .join("\n");
      handlers.onEvent(event, data);
    }
  }
}

async function parseApiResponse<T>(response: Response): Promise<ApiResponse<T> | null> {
  try {
    return (await response.json()) as ApiResponse<T>;
  } catch {
    return null;
  }
}

function formatMessage(message: string, traceId?: string, debugMessage?: string) {
  const parts = [message];
  if (traceId) {
    parts.push(`traceId=${traceId}`);
  }
  if (debugMessage) {
    parts.push(`debug=${debugMessage}`);
  }
  return parts.join(" | ");
}

export { API_BASE_URL, REFRESH_TOKEN_KEY, TOKEN_KEY };
