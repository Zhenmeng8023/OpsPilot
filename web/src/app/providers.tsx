import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { ToastProvider } from "../shared/components/ToastProvider";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 20_000
    }
  }
});

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}
