import { createTempoWalletClient } from './client.js';
import { PATH_USD, TIP20_ABI } from './config.js';
const ZERO_HASH = '0x0000000000000000000000000000000000000000000000000000000000000000';
/** Send pathUSD from the platform wallet to a consumer/provider. */
export async function processWithdrawal(privateKey, withdrawal) {
    const client = createTempoWalletClient(privateKey);
    const amount = BigInt(withdrawal.amountMicroUSD);
    try {
        const hash = await client.writeContract({
            address: PATH_USD,
            abi: TIP20_ABI,
            functionName: 'transfer',
            args: [withdrawal.toAddress, amount],
        });
        const publicClient = (await import('./client.js')).createTempoPublicClient();
        const receipt = await publicClient.waitForTransactionReceipt({ hash });
        return {
            toAddress: withdrawal.toAddress,
            amountMicroUSD: withdrawal.amountMicroUSD,
            txHash: hash,
            success: receipt.status === 'success',
            error: receipt.status !== 'success' ? 'Transaction reverted' : undefined,
        };
    }
    catch (error) {
        return {
            toAddress: withdrawal.toAddress,
            amountMicroUSD: withdrawal.amountMicroUSD,
            txHash: ZERO_HASH,
            success: false,
            error: `${error}`,
        };
    }
}
//# sourceMappingURL=withdraw.js.map