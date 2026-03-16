import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Hash, Address } from 'viem'

// Mock the client module
vi.mock('../src/client.js', () => ({
  createTempoWalletClient: vi.fn(),
  createTempoPublicClient: vi.fn(),
}))

import { createTempoWalletClient, createTempoPublicClient } from '../src/client.js'
import { jobIdToMemo, sendPayout, batchPayouts } from '../src/payout.js'
import type { PayoutRequest } from '../src/types.js'

const MOCK_PRIVATE_KEY = '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80' as `0x${string}`
const MOCK_TX_HASH = '0xdeadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678' as Hash
const MOCK_PROVIDER = '0x2222222222222222222222222222222222222222' as Address

describe('jobIdToMemo', () => {
  it('should produce consistent bytes32 for the same job ID', () => {
    const memo1 = jobIdToMemo('job-abc-123')
    const memo2 = jobIdToMemo('job-abc-123')

    expect(memo1).toBe(memo2)
    expect(memo1).toMatch(/^0x[0-9a-f]{64}$/)
  })

  it('should produce different memos for different job IDs', () => {
    const memo1 = jobIdToMemo('job-abc-123')
    const memo2 = jobIdToMemo('job-def-456')

    expect(memo1).not.toBe(memo2)
  })

  it('should always return a 32-byte hex string', () => {
    const memo = jobIdToMemo('x')
    // 0x + 64 hex chars = 66 total
    expect(memo.length).toBe(66)
    expect(memo.startsWith('0x')).toBe(true)
  })
})

describe('sendPayout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should send a successful payout', async () => {
    const mockWriteContract = vi.fn().mockResolvedValue(MOCK_TX_HASH)
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: mockWriteContract,
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'success' }),
    } as any)

    const payout: PayoutRequest = {
      providerAddress: MOCK_PROVIDER,
      amountMicroUSD: 900_000,
      jobId: 'job-test-1',
      model: 'qwen3.5-9b',
    }

    const result = await sendPayout(MOCK_PRIVATE_KEY, payout)

    expect(result.success).toBe(true)
    expect(result.txHash).toBe(MOCK_TX_HASH)
    expect(result.providerAddress).toBe(MOCK_PROVIDER)
    expect(result.amountMicroUSD).toBe(900_000)
    expect(result.memo).toMatch(/^0x[0-9a-f]{64}$/)
    expect(result.error).toBeUndefined()

    // Verify writeContract was called with correct args
    expect(mockWriteContract).toHaveBeenCalledWith(
      expect.objectContaining({
        functionName: 'transferWithMemo',
        args: [MOCK_PROVIDER, 900_000n, expect.stringMatching(/^0x/)],
      }),
    )
  })

  it('should handle transaction revert', async () => {
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockResolvedValue(MOCK_TX_HASH),
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'reverted' }),
    } as any)

    const payout: PayoutRequest = {
      providerAddress: MOCK_PROVIDER,
      amountMicroUSD: 500_000,
      jobId: 'job-test-2',
      model: 'llama3-8b',
    }

    const result = await sendPayout(MOCK_PRIVATE_KEY, payout)

    expect(result.success).toBe(false)
    expect(result.error).toBe('Transaction reverted')
  })

  it('should handle writeContract failure', async () => {
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockRejectedValue(new Error('Insufficient gas')),
    } as any)

    const payout: PayoutRequest = {
      providerAddress: MOCK_PROVIDER,
      amountMicroUSD: 100_000,
      jobId: 'job-test-3',
      model: 'test-model',
    }

    const result = await sendPayout(MOCK_PRIVATE_KEY, payout)

    expect(result.success).toBe(false)
    expect(result.error).toContain('Insufficient gas')
    expect(result.txHash).toBe('0x0000000000000000000000000000000000000000000000000000000000000000')
  })
})

describe('batchPayouts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should process multiple payouts sequentially', async () => {
    let callCount = 0
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockImplementation(() => {
        callCount++
        return Promise.resolve(
          `0x${'0'.repeat(63)}${callCount}` as Hash,
        )
      }),
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'success' }),
    } as any)

    const payouts: PayoutRequest[] = [
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 100_000, jobId: 'job-1', model: 'model-a' },
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 200_000, jobId: 'job-2', model: 'model-b' },
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 300_000, jobId: 'job-3', model: 'model-c' },
    ]

    const results = await batchPayouts(MOCK_PRIVATE_KEY, payouts)

    expect(results).toHaveLength(3)
    expect(results.every((r) => r.success)).toBe(true)
    expect(results[0].amountMicroUSD).toBe(100_000)
    expect(results[1].amountMicroUSD).toBe(200_000)
    expect(results[2].amountMicroUSD).toBe(300_000)
  })

  it('should handle empty payouts array', async () => {
    const results = await batchPayouts(MOCK_PRIVATE_KEY, [])
    expect(results).toHaveLength(0)
  })

  it('should continue batch even if one payout fails', async () => {
    let callCount = 0
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockImplementation(() => {
        callCount++
        if (callCount === 2) {
          return Promise.reject(new Error('Failed'))
        }
        return Promise.resolve(MOCK_TX_HASH)
      }),
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'success' }),
    } as any)

    const payouts: PayoutRequest[] = [
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 100_000, jobId: 'job-1', model: 'a' },
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 200_000, jobId: 'job-2', model: 'b' },
      { providerAddress: MOCK_PROVIDER, amountMicroUSD: 300_000, jobId: 'job-3', model: 'c' },
    ]

    const results = await batchPayouts(MOCK_PRIVATE_KEY, payouts)

    expect(results).toHaveLength(3)
    expect(results[0].success).toBe(true)
    expect(results[1].success).toBe(false)
    expect(results[2].success).toBe(true)
  })
})
