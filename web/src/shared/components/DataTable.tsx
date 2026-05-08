import type { ReactNode } from "react";

import { useLanguageStore } from "../../i18n/language";
import { EmptyState, ErrorState, LoadingState } from "./TableStates";

interface DataTableProps {
  children: ReactNode;
  empty?: boolean;
  emptyMessage?: ReactNode;
  error?: ReactNode;
  loading?: boolean;
}

export function DataTable({ children, empty = false, emptyMessage, error, loading = false }: DataTableProps) {
  const t = useLanguageStore((state) => state.t);

  return (
    <>
      <div className="data-table">{children}</div>
      {loading ? <LoadingState>{t("common.loading")}</LoadingState> : null}
      {!loading && empty ? <EmptyState>{emptyMessage ?? t("common.empty")}</EmptyState> : null}
      {error ? <ErrorState>{error}</ErrorState> : null}
    </>
  );
}
