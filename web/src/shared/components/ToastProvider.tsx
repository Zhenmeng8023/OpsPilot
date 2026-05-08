import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

type ToastTone = "success" | "danger" | "info";

interface ToastMessage {
  id: number;
  title: string;
  tone: ToastTone;
}

interface ToastContextValue {
  notify: (title: string, tone?: ToastTone) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastMessage[]>([]);
  const notify = useCallback((title: string, tone: ToastTone = "info") => {
    const id = Date.now() + Math.floor(Math.random() * 1000);
    setItems((current) => [...current, { id, title, tone }]);
    window.setTimeout(() => {
      setItems((current) => current.filter((item) => item.id !== id));
    }, 3200);
  }, []);
  const value = useMemo(() => ({ notify }), [notify]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite">
        {items.map((item) => (
          <div className={`toast toast-${item.tone}`} key={item.id}>
            {item.title}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return value;
}
