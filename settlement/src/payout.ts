import { createTempoWalletClient } from './client.js'
import { PATH_USD, TIP20_ABI } from './config.js'
import { type Hash, keccak256, toBytes } from 'viem'
import type { PayoutRequest, PayoutResult } from './types.js'

const ZERO_HASH = '0x0000000000000000000000000000000000000000000000000000000000000000' as Hash

/** Create a bytes32 memo from a job ID by hashing it. */
export function jobIdToMemo(jobId: string): `0x${string}` {
  return keccak256(toBytes(jobId))
}

/** Send a single pathUSD payout via transferWithMemo. */
export async function sendPayout(
  privateKey: `0x${string}`,
  payout: PayoutRequest,
): Promise<PayoutResult> {
  const client = createTempoWalletClient(privateKey)
  const memo = jobIdToMemo(payout.jobId)

  // micro-USD maps 1:1 to on-chain units (both 6 decimals)
  const amount = BigInt(payout.amountMicroUSD)

  try {
    const hash = await client.writeContract({
      address: PATH_USD,
      abi: TIP20_ABI,
      functionName: 'transferWithMemo',
      args: [payout.providerAddress, amount, memo],
    })

    // Wait for confirmation
    const publicClient = (await import('./client.js')).createTempoPublicClient()
    const receipt = await publicClient.waitForTransactionReceipt({ hash })

    return {
      providerAddress: payout.providerAddress,
      amountMicroUSD: payout.amountMicroUSD,
      txHash: hash,
      memo,
      success: receipt.status === 'success',
      error: receipt.status !== 'success' ? 'Transaction reverted' : undefined,
    }
  } catch (error) {
    return {
      providerAddress: payout.providerAddress,
      amountMicroUSD: payout.amountMicroUSD,
      txHash: ZERO_HASH,
      memo,
      success: false,
      error: `${error}`,
    }
  }
}

/**
 * Batch multiple provider payouts. Executed sequentially to avoid nonce
 * issues. In production, use Tempo's concurrent nonces (nonceKey) for
 * parallel execution.
 */
export async function batchPayouts(
  privateKey: `0x${string}`,
  payouts: PayoutRequest[],
): Promise<PayoutResult[]> {
  const results: PayoutResult[] = []

  for (const payout of payouts) {
    const result = await sendPayout(privateKey, payout)
    results.push(result)
  }

  return results
}
