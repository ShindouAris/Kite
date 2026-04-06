import { CheckIcon } from "lucide-react";
import { Button } from "../ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "../ui/card";
import { Badge } from "../ui/badge";
import { useAppSubscriptions, useBillingPlans } from "@/lib/hooks/api";
import { useAppId } from "@/lib/hooks/params";
import { useEffect, useMemo } from "react";
import { useLemonSqueezyCheckout } from "@/lib/hooks/lemonsqueezy";
import { BillingCheckoutResponse } from "@/lib/types/wire.gen";
import { formatNumber } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/router";
import { toast } from "sonner";

export default function AppPricingList() {
  const appId = useAppId();
  const queryClient = useQueryClient();
  const router = useRouter();
  const subscriptions = useAppSubscriptions();

  const activeSubscriptions = subscriptions?.filter(
    (subscription) => subscription!.status !== "expired"
  );

  const plans = useBillingPlans();

  const pricings = useMemo(() => {
    return (
      plans
        ?.filter((plan) => !plan!.hidden)
        .map((plan) => {
          return {
            ...plan!,
            current: activeSubscriptions?.some(
              (subscription) => subscription!.plan_id === plan!.id
            ),
          };
        }) ?? []
    );
  }, [activeSubscriptions, plans]);

  const checkout = useLemonSqueezyCheckout();

  useEffect(() => {
    if (!router.isReady) return;

    const paymentStatus = router.query.payment;
    if (paymentStatus === undefined) return;

    if (paymentStatus === "success") {
      toast.success("Thanh toán thành công");
      queryClient.invalidateQueries({
        queryKey: ["apps", appId, "billing", "subscriptions"],
      });
    } else if (paymentStatus === "cancel") {
      toast.message("Đã hủy thanh toán");
    } else if (paymentStatus === "error") {
      toast.error("Thanh toán thất bại");
    }

    const nextQuery = { ...router.query };
    delete nextQuery.payment;
    delete nextQuery.plan_id;
    delete nextQuery.invoice;
    router.replace({ pathname: router.pathname, query: nextQuery }, undefined, {
      shallow: true,
    });
  }, [appId, queryClient, router]);

  const submitCheckoutForm = (checkoutData: BillingCheckoutResponse) => {
    const form = document.createElement("form");
    form.action = checkoutData.action_url;
    form.method = checkoutData.method || "POST";
    form.style.display = "none";

    for (const field of checkoutData.fields) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = field.name;
      input.value = field.value;
      form.appendChild(input);
    }

    document.body.appendChild(form);
    form.submit();
    window.setTimeout(() => form.remove(), 0);
  };

  return (
    <>
      <div className="grid lg:grid-cols-2 xl:grid-cols-3 gap-8 xl:mx-16">
        {pricings.map((pricing) => (
          <Card
            key={pricing.title}
            className={
              pricing.popular
                ? "drop-shadow-xl shadow-black/10 dark:shadow-white/10"
                : "xl:my-8 "
            }
          >
            <CardHeader>
              <CardTitle className="flex item-center justify-between">
                {pricing.title}
                {pricing.popular ? (
                  <Badge variant="secondary" className="text-sm text-primary">
                    Đáng giá nhất
                  </Badge>
                ) : null}
              </CardTitle>
              <div>
                <span className="text-3xl font-bold">
                  {pricing.price.toLocaleString()}đ
                </span>
                <span className="text-muted-foreground"> /month</span>
              </div>

              <CardDescription>{pricing.description}</CardDescription>
            </CardHeader>

            <CardContent>
              <Button
                className="w-full"
                disabled={pricing.current || pricing.price === 0}
                variant={pricing.popular ? "default" : "outline"}
                onClick={() =>
                  checkout(pricing.id, (data) => {
                    submitCheckoutForm(data);
                  })
                }
              >
                {pricing.current ? "Gói hiện tại" : "Bắt đầu ngay"}
              </Button>
            </CardContent>

            <hr className="w-4/5 m-auto mb-4" />

            <CardFooter className="flex">
              <div className="space-y-4">
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">
                    {pricing.feature_max_collaborators} Cộng tác viên
                  </h3>
                </span>
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">
                    {formatNumber(pricing.feature_usage_credits_per_month)}{" "}
                    Credits / tháng
                  </h3>
                </span>
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">{pricing.feature_max_guilds} Server</h3>
                </span>
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">
                    {pricing.feature_max_commands} Lệnh & Biến
                  </h3>
                </span>
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">
                    {pricing.feature_max_event_listeners} Bộ lắng nghe sự kiện
                  </h3>
                </span>
                <span className="flex">
                  <CheckIcon className="text-green-500" />{" "}
                  <h3 className="ml-2">
                    {pricing.feature_priority_support
                      ? "Hỗ trợ ưu tiên"
                      : "Hỗ trợ cộng đồng"}
                  </h3>
                </span>
              </div>
            </CardFooter>
          </Card>
        ))}
      </div>
    </>
  );
}
