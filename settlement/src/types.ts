import type { Address, Hash } from 'viem'

/** Result of verifying an on-chain pathUSD deposit. */
export interface DepositVerification {
  verified: boolean
  txHash: Hash
  from: Address
  amount: bigint
  amountUSD: string
  amountMicroUSD: number
  blockNumber: bigint
  error?: string
}

/** A single payout request to a provider. */
export interface PayoutRequest {
  providerAddress: Address
  amountMicroUSD: number
  jobId: string
  model: string
}

/** Result of a single provider payout. */
export interface PayoutResult {
  providerAddress: Address
  amountMicroUSD: number
  txHash: Hash
  memo: `0x${string}`
  success: boolean
  error?: string
}

/** A withdrawal request from a consumer or provider. */
export interface WithdrawalRequest {
  toAddress: Address
  amountMicroUSD: number
  reason: string
}

/** Result of a withdrawal. */
export interface WithdrawalResult {
  toAddress: Address
  amountMicroUSD: number
  txHash: Hash
  success: boolean
  error?: string
}
