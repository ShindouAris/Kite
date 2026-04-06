import { generateHMAC } from "./hmac";
import { get_new_transactions } from "./mbbank";
import axios from "axios";
import axiosretry from "axios-retry";

axiosretry(axios, {retries: 3, 
    retryDelay: (retryCount) => retryCount * 2000, 
}); // Thử lại 3 lần với khoảng cách 2s nếu có lỗi khi gửi webhook

export const checkForNewTransactions = async (username: string, password: string, accountNumber: string) => {
    try {
        const newResult = await get_new_transactions(username, password, accountNumber);
        if (newResult && newResult.length > 0) {
            for (const tx of newResult) {
                // Loop qua từng giao dịch mới và gửi đến webhook
                const WEBHOOK_URL = process.env.WEBHOOK_SERVICE;
                const HMAC = process.env.HMAC;

                if (!WEBHOOK_URL || !HMAC) {
                    console.error("WEBHOOK_SERVICE URL or HMAC is not defined in environment variables.");
                    break;
                }
                try {
                    const payload = {
                        refNo: tx.refNo, // Mã tham chiếu giao dịch
                        amount: tx.creditAmount, // Tiền vào
                        transactionDate: tx.transactionDate, // Ngày giao dịch
                        postingDate: tx.postingDate, // Ngày ghi sổ
                        description: tx.addDescription, // Mô tả giao dịch
                        sender: tx.benAccountName, // Tên người gửi
                        senderAccoundNo: tx.benAccountNo // Số tài khoản người gửi
                    }
                    const generatedHMAC = generateHMAC(JSON.stringify(payload));
                    const res = await axios.post(WEBHOOK_URL, payload, {
                        headers: {
                            "X-HMAC-Signature": generatedHMAC
                        },
                        timeout: 15000 // Chờ 15s cho phản hồi từ webhook, quá sẽ gọi lại
                    })

                    if (res.status === 200) {
                        console.log(`Giao dịch ${tx.refNo} đã được gửi đến webhook thành công.`);
                    }
                } catch (error) {
                    if (error instanceof Error) {
                        console.error("Lỗi khi gửi giao dịch đến webhook:", error.message);
                    }
                    if (error instanceof axios.AxiosError) {
                        if (error.status === 400) {
                            // Phản hồi webhook đã nhận và xảy ra lỗi ở backend, ko cần thử lại
                            console.error(`Webhook đã nhận giao dịch ${tx.refNo} nhưng có lỗi đã xảy ra`);
                        }
                    }
                }
            }
        }

    } catch (error) {
        if (error instanceof Error) {
            console.error("Lỗi khi kiểm tra giao dịch mới:", error.message);
        }
        
    }
}

const INTERVAL = 1 * 60 * 1000; // Kiểm tra mỗi 1 phút
let intervalID: NodeJS.Timeout | null = null;
export const startCron = async (username: string, password: string, accountNumber: string) => {
    await checkForNewTransactions(username, password, accountNumber); // Kiểm tra ngay khi khởi động
    intervalID = setInterval(async () => {
        await checkForNewTransactions(username, password, accountNumber);
    }, INTERVAL)
}

export const stopCron = () => {
    if (intervalID) {
        clearInterval(intervalID);
        intervalID = null;
        console.log("Cron đã được dừng.");
    }
}