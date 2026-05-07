import { create } from "zustand";

import { REFRESH_TOKEN_KEY, TOKEN_KEY, request } from "../../api/request";
import type { AuthResponse, UserProfile } from "../../api/types";

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: UserProfile | null;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string, email?: string) => Promise<void>;
  logout: () => Promise<void>;
}

const USER_KEY = "opspilot.user";

function loadUser(): UserProfile | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as UserProfile;
  } catch {
    localStorage.removeItem(USER_KEY);
    return null;
  }
}

function persistSession(session: AuthResponse) {
  localStorage.setItem(TOKEN_KEY, session.accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, session.refreshToken);
  localStorage.setItem(USER_KEY, JSON.stringify(session.user));
}

function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem(TOKEN_KEY),
  refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY),
  user: loadUser(),
  login: async (username: string, password: string) => {
    const session = await request<AuthResponse>("/api/v1/auth/login", {
      method: "POST",
      skipAuth: true,
      body: JSON.stringify({ username, password })
    });
    persistSession(session);
    set({ token: session.accessToken, refreshToken: session.refreshToken, user: session.user });
  },
  register: async (username: string, password: string, email?: string) => {
    const session = await request<AuthResponse>("/api/v1/auth/register", {
      method: "POST",
      skipAuth: true,
      body: JSON.stringify({ username, password, email })
    });
    persistSession(session);
    set({ token: session.accessToken, refreshToken: session.refreshToken, user: session.user });
  },
  logout: async () => {
    const refreshToken = get().refreshToken;
    try {
      await request<{ ok: boolean }>("/api/v1/auth/logout", {
        method: "POST",
        skipAuth: true,
        body: JSON.stringify({ refreshToken })
      });
    } finally {
      clearSession();
      set({ token: null, refreshToken: null, user: null });
    }
  }
}));
