import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthStore } from "./store";

export function LoginPage() {
  const navigate = useNavigate();
  const loginDemo = useAuthStore((state) => state.loginDemo);
  const [username, setUsername] = useState("admin");

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    loginDemo(username.trim() || "admin");
    navigate("/dashboard", { replace: true });
  }

  return (
    <form className="login-card" onSubmit={handleSubmit}>
      <p className="eyebrow">T0 Scaffold</p>
      <h2>登录到 OpsPilot</h2>
      <p className="muted">当前是可运行开发骨架，认证接口会在 T1 阶段接入真实后端。</p>
      <label>
        用户名
        <input value={username} onChange={(event) => setUsername(event.target.value)} />
      </label>
      <label>
        密码
        <input value="opspilot-demo" type="password" readOnly />
      </label>
      <button type="submit">进入演示仪表盘</button>
    </form>
  );
}
