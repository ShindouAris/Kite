import { Elysia } from "elysia";
import { get_data } from "./module/mbbank";
import { config } from "dotenv";
import { startCron, stopCron } from "./module/cron";
config();

const USERNAME = process.env.MB_USERNAME;
const PASSWORD = process.env.MB_PASSWORD;
const ACCOUNT_NUMBER = process.env.MB_ACCOUNT;
const WEBHOOK_URL = process.env.WEBHOOK_SERVICE;
const HMAC = process.env.HMAC;

const app = new Elysia().get("/", () => "Hello Elysia").onStart(async ({server}) => {
  if (!USERNAME || !PASSWORD || !ACCOUNT_NUMBER || !WEBHOOK_URL || !HMAC) {
    console.error("MB_USERNAME, MB_PASSWORD, MB_ACCOUNT, WEBHOOK_URL, and HMAC must be defined in environment variables.");
    console.error(`Current values: USERNAME=${USERNAME}, PASSWORD=${PASSWORD ? "******" : undefined}, ACCOUNT_NUMBER=${ACCOUNT_NUMBER}, WEBHOOK_URL=${WEBHOOK_URL}, HMAC=${HMAC ? "******" : undefined}`);
    return;
  }
  console.log(`Elysia server is starting at ${server?.hostname}:${server?.port}...`);
  console.log("Elysia server has started. Starting cron job to check for new transactions...");
  startCron(USERNAME, PASSWORD, ACCOUNT_NUMBER);
}).onStop(() => {
  console.log("Elysia server is stopping...");
  stopCron();
}).listen(3000);

app.get("/check", async () => {
  
  try {
    const data = await get_data(USERNAME!, PASSWORD!, ACCOUNT_NUMBER!);
    if (data) {
      return data;
    } else {
      return { error: "Không thể lấy dữ liệu, vui lòng thử lại sau." };
    }
  } catch (error: any) {
    console.error("Lỗi khi xử lý yêu cầu /check:", error.message);
    return { error: "Đã xảy ra lỗi khi xử lý yêu cầu." };
  }
})






console.log(
  `🦊 Elysia is running at ${app.server?.hostname}:${app.server?.port}`
);
