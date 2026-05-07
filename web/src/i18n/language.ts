import { create } from "zustand";

export type Language = "zh" | "en";

const LANGUAGE_KEY = "opspilot.language";

const zh = {
  "common.ready": "就绪",
  "common.saving": "保存中",
  "common.loading": "加载中",
  "common.submit": "提交中...",
  "common.create": "创建",
  "common.save": "保存",
  "common.logout": "退出",
  "common.custom": "自定义",
  "common.total": "总数",
  "common.online": "在线",
  "common.offline": "离线",
  "common.disabled": "禁用",
  "common.action": "操作",
  "common.createdAt": "创建时间",
  "common.lastHeartbeat": "最近心跳",
  "common.version": "Version",
  "common.system": "System",
  "common.status": "状态",
  "nav.dashboard": "仪表盘",
  "nav.users": "用户",
  "nav.roles": "角色",
  "nav.agents": "Agent",
  "nav.scripts": "脚本",
  "nav.tasks": "任务",
  "layout.workspace": "Workspace",
  "language.toggle": "EN",
  "language.label": "切换为 English",
  "auth.heroTitle": "自动化运维与任务调度平台",
  "auth.heroDescription": "面向 Agent 接入、脚本模板、任务执行、实时日志和审计追踪的轻量级工程平台。",
  "auth.account": "OpsPilot Account",
  "auth.loginTitle": "登录 OpsPilot",
  "auth.registerTitle": "注册账号",
  "auth.description": "使用数据库中的真实账号访问控制台。",
  "auth.login": "登录",
  "auth.register": "注册",
  "auth.username": "用户名",
  "auth.email": "邮箱",
  "auth.password": "密码",
  "auth.createAccount": "创建账号",
  "auth.requestFailed": "请求失败",
  "dashboard.title": "系统运行概览",
  "dashboard.onlineAgents": "在线 Agent",
  "dashboard.todayTasks": "今日任务",
  "dashboard.failedTasks": "失败任务",
  "dashboard.recentAlerts": "最近告警",
  "dashboard.agentHint": "已接入注册与心跳",
  "dashboard.taskHint": "后续接入任务执行",
  "dashboard.failedHint": "状态机仍在方案中定义",
  "dashboard.alertHint": "后续接入监控告警",
  "dashboard.apiHealth": "API 健康检查",
  "dashboard.realtime": "实时",
  "dashboard.service": "服务",
  "dashboard.env": "环境",
  "dashboard.time": "时间",
  "dashboard.connectError": "无法连接后端，请确认 go run ./cmd/api 已启动。",
  "dashboard.devFlow": "后续开发主线",
  "users.title": "用户管理",
  "users.create": "创建用户",
  "users.list": "用户列表",
  "users.user": "用户",
  "users.status": "状态",
  "users.role": "角色",
  "users.initialPassword": "初始密码",
  "roles.title": "权限管理",
  "roles.create": "创建角色",
  "roles.code": "角色编码",
  "roles.name": "名称",
  "roles.description": "描述",
  "roles.catalog": "权限目录",
  "roles.permissions": "权限",
  "roles.builtIn": "内置角色",
  "roles.savePermissions": "保存权限",
  "agents.title": "Agent 管理",
  "agents.scanOffline": "扫描离线",
  "agents.agent": "Agent",
  "agents.host": "Host",
  "agents.hosts": "Hosts",
  "agents.ip": "IP",
  "agents.disable": "禁用",
  "agents.onlineCount": "online",
  "agents.totalCount": "total"
} as const;

const en: Record<keyof typeof zh, string> = {
  "common.ready": "Ready",
  "common.saving": "Saving",
  "common.loading": "Loading",
  "common.submit": "Submitting...",
  "common.create": "Create",
  "common.save": "Save",
  "common.logout": "Logout",
  "common.custom": "Custom",
  "common.total": "total",
  "common.online": "Online",
  "common.offline": "Offline",
  "common.disabled": "Disabled",
  "common.action": "Action",
  "common.createdAt": "Created at",
  "common.lastHeartbeat": "Last heartbeat",
  "common.version": "Version",
  "common.system": "System",
  "common.status": "Status",
  "nav.dashboard": "Dashboard",
  "nav.users": "Users",
  "nav.roles": "Roles",
  "nav.agents": "Agents",
  "nav.scripts": "Scripts",
  "nav.tasks": "Tasks",
  "layout.workspace": "Workspace",
  "language.toggle": "中",
  "language.label": "Switch to Chinese",
  "auth.heroTitle": "Automation ops and task scheduling platform",
  "auth.heroDescription": "A lightweight engineering platform for Agent access, script templates, task runs, live logs, and audit trails.",
  "auth.account": "OpsPilot Account",
  "auth.loginTitle": "Log in to OpsPilot",
  "auth.registerTitle": "Create an account",
  "auth.description": "Use a real account from the database to access the control plane.",
  "auth.login": "Log in",
  "auth.register": "Register",
  "auth.username": "Username",
  "auth.email": "Email",
  "auth.password": "Password",
  "auth.createAccount": "Create account",
  "auth.requestFailed": "Request failed",
  "dashboard.title": "System overview",
  "dashboard.onlineAgents": "Online Agents",
  "dashboard.todayTasks": "Tasks today",
  "dashboard.failedTasks": "Failed tasks",
  "dashboard.recentAlerts": "Recent alerts",
  "dashboard.agentHint": "Registration and heartbeat are connected",
  "dashboard.taskHint": "Task execution comes next",
  "dashboard.failedHint": "State machine is still defined in the plan",
  "dashboard.alertHint": "Monitoring and alerts come next",
  "dashboard.apiHealth": "API health check",
  "dashboard.realtime": "Realtime",
  "dashboard.service": "Service",
  "dashboard.env": "Environment",
  "dashboard.time": "Time",
  "dashboard.connectError": "Cannot connect to the backend. Confirm go run ./cmd/api is running.",
  "dashboard.devFlow": "Development flow",
  "users.title": "User management",
  "users.create": "Create user",
  "users.list": "User list",
  "users.user": "User",
  "users.status": "Status",
  "users.role": "Role",
  "users.initialPassword": "Initial password",
  "roles.title": "Permission management",
  "roles.create": "Create role",
  "roles.code": "Role code",
  "roles.name": "Name",
  "roles.description": "Description",
  "roles.catalog": "Permission catalog",
  "roles.permissions": "Permissions",
  "roles.builtIn": "Built-in role",
  "roles.savePermissions": "Save permissions",
  "agents.title": "Agent management",
  "agents.scanOffline": "Scan offline",
  "agents.agent": "Agent",
  "agents.host": "Host",
  "agents.hosts": "Hosts",
  "agents.ip": "IP",
  "agents.disable": "Disable",
  "agents.onlineCount": "online",
  "agents.totalCount": "total"
};

type MessageKey = keyof typeof zh;

const messages: Record<Language, Record<MessageKey, string>> = { zh, en };

interface LanguageState {
  language: Language;
  setLanguage: (language: Language) => void;
  toggleLanguage: () => void;
  t: (key: MessageKey) => string;
}

export const useLanguageStore = create<LanguageState>((set, get) => {
  const initialLanguage = resolveInitialLanguage();
  applyDocumentLanguage(initialLanguage);

  return {
    language: initialLanguage,
    setLanguage: (language) => {
      localStorage.setItem(LANGUAGE_KEY, language);
      applyDocumentLanguage(language);
      set({ language, t: (key) => messages[language][key] });
    },
    toggleLanguage: () => {
      const next = get().language === "zh" ? "en" : "zh";
      get().setLanguage(next);
    },
    t: (key) => messages[initialLanguage][key]
  };
});

function resolveInitialLanguage(): Language {
  const saved = localStorage.getItem(LANGUAGE_KEY);
  if (saved === "zh" || saved === "en") {
    return saved;
  }
  return navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
}

function applyDocumentLanguage(language: Language) {
  document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
}
