import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Hash, Address } from 'viem'

// Mock the client module before importing deposit
vi.mock('../src/client.js', () => ({
  createTempoPublicClient: vi.fn(),
}))

import { createTempoPublicClient } from '../src/client.js'
import { verifyDeposit, getPathUSDBalance } from '../src/deposit.js'
import { PATH_USD } from '../src/config.js'

const PLATFORM_WALLET = '0x0000000000000000000000000000000000000000'
const MOCK_TX_HASH = '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890' as Hash
const MOCK_SENDER = '0x1111111111111111111111111111111111111111' as Address

// Transfer event signature
const TRANSFER_EVENT_SIG = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef'

function padAddress(addr: string): `0x${string}` {
  return ('0x' + addr.slice(2).padStart(64, '0')) as `0x${string}`
}

describe('verifyDeposit', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should verify a successful deposit', async () => {
    const mockReceipt = {
      status: 'success' as const,
      blockNumber: 12345n,
      logs: [
        {
          address: PATH_USD,
          topics: [
            TRANSFER_EVENT_SIG,
            padAddress(MOCK_SENDER),
            padAddress(PLATFORM_WALLET),
          ] as [`0x${string}`, ...`0x${string}`[]],
          data: '0x00000000000000000000000000000000000000000000000000000000000f4240' as `0x${string}`, // 1,000,000 = $1.00
          blockNumber: 12345n,
          blockHash: '0x0' as `0x${string}`,
          transactionHash: MOCK_TX_HASH,
          transactionIndex: 0,
          logIndex: 0,
          removed: false,
        },
      ],
    }

    vi.mocked(createTempoPublicClient).mockReturnValue({
      getTransactionReceipt: vi.fn().mockResolvedValue(mockReceipt),
    } as any)

    const result = await verifyDeposit(MOCK_TX_HASH)

    expect(result.verified).toBe(true)
    expect(result.txHash).toBe(MOCK_TX_HASH)
    expect(result.amount).toBe(1_000_000n)
    expect(result.amountUSD).toBe('1')
    expect(result.amountMicroUSD).toBe(1_000_000)
    expect(result.blockNumber).toBe(12345n)
    expect(result.error).toBeUndefined()
  })

  it('should reject a failed transaction', async () => {
    const mockReceipt = {
      status: 'reverted' as const,
      blockNumber: 12345n,
      logs: [],
    }

    vi.mocked(createTempoPublicClient).mockReturnValue({
      getTransactionReceipt: vi.fn().mockResolvedValue(mockReceipt),
    } as any)

    const result = await verifyDeposit(MOCK_TX_HASH)

    expect(result.verified).toBe(false)
    expect(result.error).toBe('Transaction failed')
  })

  it('should reject when no Transfer event to platform wallet', async () => {
    const otherAddress = '0x9999999999999999999999999999999999999999' as Address

    const mockReceipt = {
      status: 'success' as const,
      blockNumber: 12345n,
      logs: [
        {
          address: PATH_USD,
          topics: [
            TRANSFER_EVENT_SIG,
            padAddress(MOCK_SENDER),
            padAddress(otherAddress),
          ] as [`0x${string}`, ...`0x${string}`[]],
          data: '0x00000000000000000000000000000000000000000000000000000000000f4240' as `0x${string}`,
          blockNumber: 12345n,
          blockHash: '0x0' as `0x${string}`,
          transactionHash: MOCK_TX_HASH,
          transactionIndex: 0,
          logIndex: 0,
          removed: false,
        },
      ],
    }

    vi.mocked(createTempoPublicClient).mockReturnValue({
      getTransactionReceipt: vi.fn().mockResolvedValue(mockReceipt),
    } as any)

    const result = await verifyDeposit(MOCK_TX_HASH)

    expect(result.verified).toBe(false)
    expect(result.error).toBe('No pathUSD transfer to platform wallet found')
  })

  it('should handle RPC errors gracefully', async () => {
    vi.mocked(createTempoPublicClient).mockReturnValue({
      getTransactionReceipt: vi.fn().mockRejectedValue(new Error('RPC connection failed')),
    } as any)

    const result = await verifyDeposit(MOCK_TX_HASH)

    expect(result.verified).toBe(false)
    expect(result.error).toContain('Failed to verify')
    expect(result.error).toContain('RPC connection failed')
  })

  it('should parse amount with 6 decimals correctly', async () => {
    // 10,500,000 = $10.50
    const mockReceipt = {
      status: 'success' as const,
      blockNumber: 99n,
      logs: [
        {
          address: PATH_USD,
          topics: [
            TRANSFER_EVENT_SIG,
            padAddress(MOCK_SENDER),
            padAddress(PLATFORM_WALLET),
          ] as [`0x${string}`, ...`0x${string}`[]],
          data: '0x0000000000000000000000000000000000000000000000000000000000a05fc0' as `0x${string}`, // 10,518,464
          blockNumber: 99n,
          blockHash: '0x0' as `0x${string}`,
          transactionHash: MOCK_TX_HASH,
          transactionIndex: 0,
          logIndex: 0,
          removed: false,
        },
      ],
    }

    vi.mocked(createTempoPublicClient).mockReturnValue({
      getTransactionReceipt: vi.fn().mockResolvedValue(mockReceipt),
    } as any)

    const result = await verifyDeposit(MOCK_TX_HASH)

    expect(result.verified).toBe(true)
    expect(result.amountMicroUSD).toBe(Number(result.amount))
  })
})

describe('getPathUSDBalance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should return balance and formatted USD', async () => {
    vi.mocked(createTempoPublicClient).mockReturnValue({
      readContract: vi.fn().mockResolvedValue(5_000_000n),
    } as any)

    const result = await getPathUSDBalance(MOCK_SENDER)

    expect(result.balance).toBe(5_000_000n)
    expect(result.balanceUSD).toBe('5')
  })
})
