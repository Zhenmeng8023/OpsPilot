import type { ReactNode } from "react";

import { useLanguageStore } from "../i18n/language";

export function AuthLayout({ children }: { children: ReactNode }) {
  const t = useLanguageStore((state) => state.t);
  const toggleLanguage = useLanguageStore((state) => state.toggleLanguage);

  return (
    <main className="auth-shell">
      <section className="auth-hero">
        <div className="auth-language">
          <button type="button" className="language-button" aria-label={t("language.label")} onClick={toggleLanguage}>
            {t("language.toggle")}
          </button>
        </div>
        <p className="eyebrow">{t("layout.controlPlane")}</p>
        <h1>{t("auth.heroTitle")}</h1>
        <p>{t("auth.heroDescription")}</p>
        <div className="hero-grid">
          <span>Agent</span>
          <span>Task Runs</span>
          <span>SSE Logs</span>
          <span>Audit</span>
        </div>
      </section>
      <section className="auth-panel">{children}</section>
    </main>
  );
}
