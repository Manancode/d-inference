import { createTempoPublicClient } from './client.js';
import { PATH_USD, TIP20_ABI, PATH_USD_DECIMALS, config } from './config.js';
import { formatUnits } from 'viem';
const ZERO_ADDRESS = '0x0000000000000000000000000000000000000000';
const ZERO_HASH = '0x0000000000000000000000000000000000000000000000000000000000000000';
/**
 * Verify a pathUSD transfer to the platform wallet by inspecting the
 * transaction receipt for a Transfer event to our platform address.
 */
export async function verifyDeposit(txHash) {
    const client = createTempoPublicClient();
    try {
        const receipt = await client.getTransactionReceipt({ hash: txHash });
        if (receipt.status !== 'success') {
            return {
                verified: false,
                txHash,
                from: ZERO_ADDRESS,
                amount: 0n,
                amountUSD: '0',
                amountMicroUSD: 0,
                blockNumber: 0n,
                error: 'Transaction failed',
            };
        }
        // Find the Transfer event targeting the platform wallet
        const transferLogs = receipt.logs.filter((log) => log.address.toLowerCase() === PATH_USD.toLowerCase());
        for (const log of transferLogs) {
            // Transfer event topics: [event_sig, from, to], data: [value]
            if (log.topics.length >= 3) {
                const to = ('0x' + log.topics[2].slice(26));
                if (to.toLowerCase() === config.platformWallet.toLowerCase()) {
                    const from = ('0x' + log.topics[1].slice(26));
                    const amount = BigInt(log.data);
                    const amountUSD = formatUnits(amount, PATH_USD_DECIMALS);
                    // pathUSD 6 decimals maps 1:1 to micro-USD
                    const amountMicroUSD = Number(amount);
                    return {
                        verified: true,
                        txHash,
                        from,
                        amount,
                        amountUSD,
                        amountMicroUSD,
                        blockNumber: receipt.blockNumber,
                    };
                }
            }
        }
        return {
            verified: false,
            txHash,
            from: ZERO_ADDRESS,
            amount: 0n,
            amountUSD: '0',
            amountMicroUSD: 0,
            blockNumber: receipt.blockNumber,
            error: 'No pathUSD transfer to platform wallet found',
        };
    }
    catch (error) {
        return {
            verified: false,
            txHash,
            from: ZERO_ADDRESS,
            amount: 0n,
            amountUSD: '0',
            amountMicroUSD: 0,
            blockNumber: 0n,
            error: `Failed to verify: ${error}`,
        };
    }
}
/** Check the pathUSD balance of an address on-chain. */
export async function getPathUSDBalance(address) {
    const client = createTempoPublicClient();
    const balance = await client.readContract({
        address: PATH_USD,
        abi: TIP20_ABI,
        functionName: 'balanceOf',
        args: [address],
    });
    return {
        balance,
        balanceUSD: formatUnits(balance, PATH_USD_DECIMALS),
    };
}
//# sourceMappingURL=deposit.js.map