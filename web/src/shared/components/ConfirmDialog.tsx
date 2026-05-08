import type { ReactNode } from "react";

interface ConfirmDialogProps {
  open: boolean;
  title: ReactNode;
  message?: ReactNode;
  confirmLabel: ReactNode;
  cancelLabel: ReactNode;
  danger?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  cancelLabel,
  danger = false,
  onCancel,
  onConfirm
}: ConfirmDialogProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <section className="confirm-dialog" role="dialog" aria-modal="true">
        <h3>{title}</h3>
        {message ? <p>{message}</p> : null}
        <div className="confirm-actions">
          <button type="button" onClick={onCancel}>{cancelLabel}</button>
          <button className={danger ? "danger-button" : "ghost-button"} type="button" onClick={onConfirm}>{confirmLabel}</button>
        </div>
      </section>
    </div>
  );
}
