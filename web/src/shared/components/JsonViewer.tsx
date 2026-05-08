import { useLanguageStore } from "../../i18n/language";

interface JsonViewerProps {
  value?: unknown;
  emptyLabel?: string;
}

export function JsonViewer({ value, emptyLabel }: JsonViewerProps) {
  const t = useLanguageStore((state) => state.t);
  const content = formatJsonValue(value);

  return (
    <pre className="json-block" data-empty={!content}>
      {content || emptyLabel || t("common.empty")}
    </pre>
  );
}

function formatJsonValue(value?: unknown) {
  if (value === undefined || value === null || value === "") {
    return "";
  }
  if (typeof value !== "string") {
    return JSON.stringify(value, null, 2);
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed) as unknown, null, 2);
  } catch {
    return trimmed;
  }
}
