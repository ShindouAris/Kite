import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { useAppSubscriptionsQuery } from "@/lib/api/queries";
import { useAppId } from "@/lib/hooks/params";
import { BillingCheckoutResponse } from "@/lib/types/wire.gen";
import { formatNumber } from "@/lib/utils";
import { CheckCircle2Icon, Clock3Icon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

type PaymentState = "pending" | "success" | "expired";
const POLL_BASE_INTERVAL_MS = 8000;
const POLL_MAX_INTERVAL_MS = 30000;

interface QRCodePaymentModalProps {
  open: boolean;
  checkout: BillingCheckoutResponse | null;
  planTitle?: string;
  planProductId?: string;
  onClose: () => void;
  onConfirmed?: () => void;
}

export default function QRCodePaymentModal({
  open,
  checkout,
  planTitle,
  planProductId,
  onClose,
  onConfirmed,
}: QRCodePaymentModalProps) {
  const appId = useAppId();
  const subscriptionsQuery = useAppSubscriptionsQuery(appId);
  const { data: subscriptionsData, isFetching, refetch } = subscriptionsQuery;
  const [paymentState, setPaymentState] = useState<PaymentState>("pending");
  const confirmedRef = useRef(false);
  const pollAttemptRef = useRef(0);

  const expiresAtTime = checkout ? new Date(checkout.expires_at).getTime() : 0;

  const qrUrl = useMemo(() => {
    if (!checkout) return "";

    const amount = encodeURIComponent(String(checkout.amount));
    const transferContent = encodeURIComponent(checkout.transfer_content);
    const accountName = encodeURIComponent(checkout.bank_name);

    return `https://img.vietqr.io/image/MB-${checkout.account_number}-qr-only.png?amount=${amount}&addInfo=${transferContent}&accountName=${accountName}`;
  }, [checkout]);

  useEffect(() => {
    if (!open) {
      setPaymentState("pending");
      confirmedRef.current = false;
      pollAttemptRef.current = 0;
      return;
    }

    if (!checkout || !expiresAtTime) return;

    if (Date.now() > expiresAtTime) {
      setPaymentState("expired");
      return;
    }

    const expiryTimer = window.setInterval(() => {
      if (Date.now() > expiresAtTime) {
        setPaymentState("expired");
      }
    }, 1000);

    return () => {
      window.clearInterval(expiryTimer);
    };
  }, [checkout, expiresAtTime, open]);

  useEffect(() => {
    if (!open || paymentState !== "pending") return;

    let stopped = false;
    let timer: number | undefined;

    const schedulePoll = () => {
      const delay = Math.min(
        POLL_BASE_INTERVAL_MS * Math.max(1, pollAttemptRef.current + 1),
        POLL_MAX_INTERVAL_MS
      );

      timer = window.setTimeout(async () => {
        if (stopped || document.hidden) {
          schedulePoll();
          return;
        }

        if (!isFetching) {
          await refetch();
          pollAttemptRef.current += 1;
        }

        schedulePoll();
      }, delay);
    };

    schedulePoll();

    return () => {
      stopped = true;
      if (timer) {
        window.clearTimeout(timer);
      }
    };
  }, [
    open,
    paymentState,
    isFetching,
    refetch,
  ]);

  useEffect(() => {
    if (!open || paymentState !== "pending" || !planProductId) return;
    if (!subscriptionsData?.success) return;

    const hasActivePlan = subscriptionsData.data?.some(
      (subscription) =>
        subscription &&
        subscription.status !== "expired" &&
        subscription.lemonsqueezy_product_id === planProductId
    );

    if (!hasActivePlan || confirmedRef.current) return;

    confirmedRef.current = true;
    setPaymentState("success");
    onConfirmed?.();

    const closeTimer = window.setTimeout(() => {
      onClose();
    }, 1200);

    return () => {
      window.clearTimeout(closeTimer);
    };
  }, [
    onClose,
    onConfirmed,
    open,
    paymentState,
    planProductId,
    subscriptionsData,
  ]);

  if (!checkout) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Thanh toán bằng QR</DialogTitle>
          <DialogDescription>
            Vui lòng quét QR để thanh toán cho gói {planTitle ?? "Premium"}.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-5 sm:grid-cols-[220px_1fr]">
          <div className="rounded-lg border p-2">
            <img src={qrUrl} alt="QR thanh toan" className="h-full w-full" />
          </div>

          <div className="space-y-3 text-sm">
            <div className="font-medium text-base">Thông tin chuyển khoản</div>
            <div>
              <span className="text-muted-foreground">Người nhận: </span>
              <span>{checkout.bank_name}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Số tài khoản: </span>
              <span>{checkout.account_number}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Số tiền: </span>
              <span className="font-semibold">{formatNumber(checkout.amount)} VND</span>
            </div>
            <div>
              <span className="text-muted-foreground">Nội dung: </span>
              <span className="break-all">{checkout.transfer_content}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Hết hạn lúc: </span>
              <span>{new Date(checkout.expires_at).toLocaleString("vi-VN")}</span>
            </div>
          </div>
        </div>

        <Separator />

        {paymentState === "success" ? (
          <div className="flex items-center gap-2 rounded-md bg-green-50 px-3 py-2 text-green-700 dark:bg-green-950/40 dark:text-green-300">
            <CheckCircle2Icon className="h-4 w-4" />
            Đã xác nhận thanh toán. Đang đóng hộp thoại...
          </div>
        ) : paymentState === "expired" ? (
          <div className="flex items-center gap-2 rounded-md bg-amber-50 px-3 py-2 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">
            <Clock3Icon className="h-4 w-4" />
            Mã thanh toán đã hết hạn. Vui lòng tạo checkout mới.
          </div>
        ) : (
          <div className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
            Hệ thống đang tự động kiểm tra giao dịch. Sau khi thanh toán thành công, hộp thoại sẽ tự động đóng.
          </div>
        )}

        <DialogFooter>
          {paymentState !== "success" && (
            <Button
              variant="outline"
              onClick={() => refetch()}
              disabled={isFetching}
            >
                Tôi đã thanh toán, kiểm tra ngay
            </Button>
          )}
          <Button onClick={onClose}>Đóng</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}