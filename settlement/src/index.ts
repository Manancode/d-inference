import express from 'express'
import { config } from './config.js'
import { verifyDeposit, getPathUSDBalance } from './deposit.js'
import { batchPayouts } from './payout.js'
import { processWithdrawal } from './withdraw.js'
import type { PayoutRequest } from './types.js'
import type { Address, Hash } from 'viem'

const app = express()
app.use(express.json())

// Health check
app.get('/health', (_req, res) => {
  res.json({ status: 'ok', chain: config.chain })
})

// Verify an on-chain deposit
app.post('/v1/settlement/verify-deposit', async (req, res) => {
  const { tx_hash } = req.body
  if (!tx_hash) {
    res.status(400).json({ error: 'tx_hash required' })
    return
  }

  const result = await verifyDeposit(tx_hash as Hash)

  if (result.verified) {
    // TODO: Call coordinator to credit the consumer's ledger balance
    // POST coordinator/v1/payments/deposit with verified amount
  }

  // Serialize BigInt values for JSON
  res.json({
    ...result,
    amount: result.amount.toString(),
    blockNumber: result.blockNumber.toString(),
  })
})

// Check pathUSD balance of an address
app.get('/v1/settlement/balance/:address', async (req, res) => {
  const result = await getPathUSDBalance(req.params.address as Address)
  res.json({
    address: req.params.address,
    balance: result.balance.toString(),
    balanceUSD: result.balanceUSD,
  })
})

// Process pending provider payouts
app.post('/v1/settlement/payouts', async (req, res) => {
  const { payouts, private_key } = req.body as {
    payouts: PayoutRequest[]
    private_key: string
  }
  if (!payouts || !private_key) {
    res.status(400).json({ error: 'payouts and private_key required' })
    return
  }

  const results = await batchPayouts(private_key as `0x${string}`, payouts)
  res.json({
    results: results.map((r) => ({
      ...r,
    })),
    settled: results.filter((r) => r.success).length,
    failed: results.filter((r) => !r.success).length,
  })
})

// Process a withdrawal
app.post('/v1/settlement/withdraw', async (req, res) => {
  const { to_address, amount_micro_usd, reason, private_key } = req.body
  if (!to_address || !amount_micro_usd || !private_key) {
    res.status(400).json({
      error: 'to_address, amount_micro_usd, and private_key required',
    })
    return
  }

  const result = await processWithdrawal(private_key as `0x${string}`, {
    toAddress: to_address,
    amountMicroUSD: amount_micro_usd,
    reason: reason || 'withdrawal',
  })
  res.json(result)
})

app.listen(config.port, () => {
  console.log(`DGInf settlement service listening on port ${config.port}`)
  console.log(`Chain: ${config.chain}`)
  console.log(`RPC: ${config.rpcUrl}`)
})

export { app }
