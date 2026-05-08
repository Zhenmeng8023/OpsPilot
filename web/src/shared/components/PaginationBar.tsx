import { useLanguageStore } from "../../i18n/language";

interface PaginationBarProps {
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

export function PaginationBar({ total, page, pageSize, onPageChange }: PaginationBarProps) {
  const t = useLanguageStore((state) => state.t);

  return (
    <div className="toolbar-row pagination-bar">
      <span className="pagination-summary">
        {total} {t("common.total")}
      </span>
      <button type="button" disabled={page <= 1} onClick={() => onPageChange(Math.max(1, page - 1))}>
        {t("common.previous")}
      </button>
      <span>{t("common.page", { page })}</span>
      <button type="button" disabled={page * pageSize >= total} onClick={() => onPageChange(page + 1)}>
        {t("common.next")}
      </button>
    </div>
  );
}
