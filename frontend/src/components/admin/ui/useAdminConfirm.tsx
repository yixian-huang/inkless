import { useCallback, useRef, useState, type ReactNode } from "react";
import AdminConfirmDialog from "./AdminConfirmDialog";

export type AdminConfirmOptions = {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
};

type Pending = AdminConfirmOptions & {
  resolve: (value: boolean) => void;
};

/**
 * Imperative confirm helper for admin pages.
 *
 * const { confirm, confirmDialog } = useAdminConfirm();
 * if (!(await confirm({ title: "删除", message: "...", danger: true }))) return;
 * return (<>{confirmDialog}...</>);
 */
export function useAdminConfirm() {
  const [pending, setPending] = useState<Pending | null>(null);
  const [loading, setLoading] = useState(false);
  const loadingRef = useRef(false);

  const confirm = useCallback((options: AdminConfirmOptions) => {
    return new Promise<boolean>((resolve) => {
      setPending({ ...options, resolve });
    });
  }, []);

  const close = useCallback((value: boolean) => {
    if (loadingRef.current) return;
    setPending((current) => {
      current?.resolve(value);
      return null;
    });
  }, []);

  const runWithLoading = useCallback(async (fn: () => void | Promise<void>) => {
    loadingRef.current = true;
    setLoading(true);
    try {
      await fn();
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, []);

  const confirmDialog: ReactNode = (
    <AdminConfirmDialog
      open={Boolean(pending)}
      title={pending?.title ?? ""}
      message={pending?.message ?? ""}
      confirmLabel={pending?.confirmLabel}
      cancelLabel={pending?.cancelLabel}
      danger={pending?.danger}
      loading={loading}
      onCancel={() => close(false)}
      onConfirm={() => close(true)}
    />
  );

  return { confirm, confirmDialog, runWithLoading, setConfirmLoading: setLoading };
}
