import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Hash, Address } from 'viem'

// Mock the client module
vi.mock('../src/client.js', () => ({
  createTempoWalletClient: vi.fn(),
  createTempoPublicClient: vi.fn(),
}))

import { createTempoWalletClient, createTempoPublicClient } from '../src/client.js'
import { processWithdrawal } from '../src/withdraw.js'

const MOCK_PRIVATE_KEY = '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80' as `0x${string}`
const MOCK_TX_HASH = '0xfeedface1234567890abcdef1234567890abcdef1234567890abcdef12345678' as Hash
const MOCK_CONSUMER = '0x3333333333333333333333333333333333333333' as Address

describe('processWithdrawal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should process a successful withdrawal', async () => {
    const mockWriteContract = vi.fn().mockResolvedValue(MOCK_TX_HASH)
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: mockWriteContract,
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'success' }),
    } as any)

    const result = await processWithdrawal(MOCK_PRIVATE_KEY, {
      toAddress: MOCK_CONSUMER,
      amountMicroUSD: 5_000_000,
      reason: 'consumer_withdrawal',
    })

    expect(result.success).toBe(true)
    expect(result.txHash).toBe(MOCK_TX_HASH)
    expect(result.toAddress).toBe(MOCK_CONSUMER)
    expect(result.amountMicroUSD).toBe(5_000_000)
    expect(result.error).toBeUndefined()

    // Verify transfer was called with correct args
    expect(mockWriteContract).toHaveBeenCalledWith(
      expect.objectContaining({
        functionName: 'transfer',
        args: [MOCK_CONSUMER, 5_000_000n],
      }),
    )
  })

  it('should handle a reverted withdrawal', async () => {
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockResolvedValue(MOCK_TX_HASH),
    } as any)

    vi.mocked(createTempoPublicClient).mockReturnValue({
      waitForTransactionReceipt: vi.fn().mockResolvedValue({ status: 'reverted' }),
    } as any)

    const result = await processWithdrawal(MOCK_PRIVATE_KEY, {
      toAddress: MOCK_CONSUMER,
      amountMicroUSD: 5_000_000,
      reason: 'consumer_withdrawal',
    })

    expect(result.success).toBe(false)
    expect(result.error).toBe('Transaction reverted')
  })

  it('should handle writeContract failure', async () => {
    vi.mocked(createTempoWalletClient).mockReturnValue({
      writeContract: vi.fn().mockRejectedValue(new Error('Insufficient balance')),
    } as any)

    const result = await processWithdrawal(MOCK_PRIVATE_KEY, {
      toAddress: MOCK_CONSUMER,
      amountMicroUSD: 100_000_000,
      reason: 'provider_withdrawal',
    })

    expect(result.success).toBe(false)
    expect(result.error).toContain('Insufficient balance')
    expect(result.txHash).toBe('0x0000000000000000000000000000000000000000000000000000000000000000')
  })
})
