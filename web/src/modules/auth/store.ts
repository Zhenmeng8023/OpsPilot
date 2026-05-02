import { create } from "zustand";

import { TOKEN_KEY } from "../../api/request";

interface UserProfile {
  id: string;
  username: string;
  roles: string[];
}

interface AuthState {
  token: string | null;
  user: UserProfile | null;
  loginDemo: (username: string) => void;
  logout: () => void;
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

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem(TOKEN_KEY),
  user: loadUser(),
  loginDemo: (username: string) => {
    const user: UserProfile = {
      id: "demo-user",
      username,
      roles: ["admin"]
    };
    const token = "demo-token";
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(USER_KEY, JSON.stringify(user));
    set({ token, user });
  },
  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    set({ token: null, user: null });
  }
}));
