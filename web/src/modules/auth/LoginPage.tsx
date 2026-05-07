import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthStore } from "./store";

type Mode = "login" | "register";

export function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const register = useAuthStore((state) => state.register);
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
      setError(err instanceof Error ? err.message : "请求失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="login-card" onSubmit={handleSubmit}>
      <p className="eyebrow">OpsPilot Account</p>
      <h2>{mode === "login" ? "登录 OpsPilot" : "注册账号"}</h2>
      <p className="muted">使用数据库中的真实账号访问控制台。</p>

      <div className="auth-mode" role="tablist" aria-label="auth mode">
        <button type="button" className={mode === "login" ? "active" : ""} onClick={() => setMode("login")}>
          登录
        </button>
        <button type="button" className={mode === "register" ? "active" : ""} onClick={() => setMode("register")}>
          注册
        </button>
      </div>

      <label>
        用户名
        <input value={username} autoComplete="username" onChange={(event) => setUsername(event.target.value)} />
      </label>

      {mode === "register" ? (
        <label>
          邮箱
          <input value={email} autoComplete="email" onChange={(event) => setEmail(event.target.value)} />
        </label>
      ) : null}

      <label>
        密码
        <input
          value={password}
          type="password"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          onChange={(event) => setPassword(event.target.value)}
        />
      </label>

      {error ? <p className="form-error">{error}</p> : null}

      <button type="submit" disabled={submitting}>
        {submitting ? "提交中..." : mode === "login" ? "登录" : "创建账号"}
      </button>
    </form>
  );
}
