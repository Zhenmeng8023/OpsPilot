import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useLanguageStore } from "../../i18n/language";
import { useAuthStore } from "./store";

type Mode = "login" | "register";

export function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const register = useAuthStore((state) => state.register);
  const t = useLanguageStore((state) => state.t);
  const [mode, setMode] = useState<Mode>("login");
  const [username, setUsername] = useState("admin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("Admin@123456");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      if (mode === "login") {
        await login(username.trim(), password);
      } else {
        await register(username.trim(), password, email.trim() || undefined);
      }
      navigate("/dashboard", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.requestFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="login-card" onSubmit={handleSubmit}>
      <p className="eyebrow">{t("auth.account")}</p>
      <h2>{mode === "login" ? t("auth.loginTitle") : t("auth.registerTitle")}</h2>
      <p className="muted">{t("auth.description")}</p>

      <div className="auth-mode" role="tablist" aria-label="auth mode">
        <button type="button" className={mode === "login" ? "active" : ""} onClick={() => setMode("login")}>
          {t("auth.login")}
        </button>
        <button type="button" className={mode === "register" ? "active" : ""} onClick={() => setMode("register")}>
          {t("auth.register")}
        </button>
      </div>

      <label>
        {t("auth.username")}
        <input value={username} autoComplete="username" onChange={(event) => setUsername(event.target.value)} />
      </label>

      {mode === "register" ? (
        <label>
          {t("auth.email")}
          <input value={email} autoComplete="email" onChange={(event) => setEmail(event.target.value)} />
        </label>
      ) : null}

      <label>
        {t("auth.password")}
        <input
          value={password}
          type="password"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          onChange={(event) => setPassword(event.target.value)}
        />
      </label>

      {error ? <p className="form-error">{error}</p> : null}

      <button type="submit" disabled={submitting}>
        {submitting ? t("common.submit") : mode === "login" ? t("auth.login") : t("auth.createAccount")}
      </button>
    </form>
  );
}
