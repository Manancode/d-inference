import { createPublicClient, createWalletClient, http } from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { config, TEMPO_CHAINS } from './config.js'

/** Public client for reading Tempo blockchain state. */
export function createTempoPublicClient() {
  const chain = TEMPO_CHAINS[config.chain]
  return createPublicClient({
    chain,
    transport: http(config.rpcUrl),
  })
}

/** Wallet client for sending transactions on Tempo (requires private key). */
export function createTempoWalletClient(privateKey: `0x${string}`) {
  const chain = TEMPO_CHAINS[config.chain]
  const account = privateKeyToAccount(privateKey)
  return createWalletClient({
    account,
    chain,
    transport: http(config.rpcUrl),
  })
}
