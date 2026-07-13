"use client";

import { createContext, useCallback, useContext, useMemo, useState } from "react";

type ToastTone = "success" | "error" | "info";
type Toast = { id: string; tone: ToastTone; title: string; detail?: string };
type ToastInput = Omit<Toast, "id">;

const ToastContext = createContext<{ pushToast: (toast: ToastInput) => void }>({ pushToast: () => {} });

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const pushToast = useCallback((input: ToastInput) => {
    const id = crypto.randomUUID();
    setToasts((items) => [{ id, ...input }, ...items].slice(0, 4));
    window.setTimeout(() => {
      setToasts((items) => items.filter((toast) => toast.id !== id));
    }, input.tone === "error" ? 8000 : 4500);
  }, []);

  const value = useMemo(() => ({ pushToast }), [pushToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed bottom-4 right-4 z-[60] grid w-[min(24rem,calc(100vw-2rem))] gap-2">
        {toasts.map((toast) => (
          <div key={toast.id} className={`rounded-md border bg-white p-4 shadow-panel ${toneClass(toast.tone)}`}>
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm font-semibold text-ink">{toast.title}</p>
                {toast.detail ? <p className="mt-1 text-sm leading-5 text-ink/60">{toast.detail}</p> : null}
              </div>
              <button onClick={() => setToasts((items) => items.filter((item) => item.id !== toast.id))} className="rounded px-2 text-sm text-ink/45 hover:bg-ink/[0.04]">x</button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  return useContext(ToastContext);
}

function toneClass(tone: ToastTone) {
  if (tone === "success") return "border-moss/30";
  if (tone === "error") return "border-coral/40";
  return "border-ink/15";
}
