import { useCallback } from "react";
import {
  useAppSubscriptionManageMutation,
  useCheckoutCreateMutation,
} from "../api/mutations";
import { useAppId } from "./params";
import { toast } from "sonner";
import { BillingCheckoutResponse } from "../types/wire.gen";

type CheckoutSuccessHandler = (checkout: BillingCheckoutResponse) => void;

export function useLemonSqueezyCheckout() {
  const appId = useAppId();
  const checkoutMutation = useCheckoutCreateMutation(appId);

  return useCallback(
    (planId: string, onSuccess?: CheckoutSuccessHandler) => {
      checkoutMutation.mutate(
        {
          plan_id: planId,
        },
        {
          onSuccess(res) {
            if (res.success) {
              onSuccess?.(res.data);
            } else {
              toast.error(
                `Tạo thanh toán thất bại: ${res.error.message} ${res.error.code}`
              );
            }
          },
        }
      );
    },
    [checkoutMutation]
  );
}

export function useLemonSqueezyCustomerPortal(subscriptionId: string) {
  const manageMutation = useAppSubscriptionManageMutation(subscriptionId);

  return useCallback(() => {
    manageMutation.mutate(undefined, {
      onSuccess(res) {
        if (res.success) {
          window.location.href = res.data.customer_portal_url;
        } else {
          toast.error(
            `Quản lý đăng ký thất bại: ${res.error.message} ${res.error.code}`
          );
        }
      },
    });
  }, [manageMutation]);
}

export function useLemonSqueezyUpdatePaymentMethod(subscriptionId: string) {
  const manageMutation = useAppSubscriptionManageMutation(subscriptionId);

  return useCallback(() => {
    manageMutation.mutate(undefined, {
      onSuccess(res) {
        if (res.success) {
          (window as any).LemonSqueezy.Url.Open(
            res.data.update_payment_method_url
          );
        } else {
          toast.error(
            `Quản lý đăng ký thất bại: ${res.error.message} ${res.error.code}`
          );
        }
      },
    });
  }, [manageMutation]);
}
