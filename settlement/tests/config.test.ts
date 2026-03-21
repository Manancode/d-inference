import { describe, it, expect } from 'vitest'
import { PATH_USD, PATH_USD_DECIMALS, TEMPO_CHAINS, config } from '../src/config.js'

describe('config', () => {
  it('should have correct pathUSD address', () => {
    expect(PATH_USD).toBe('0x20C0000000000000000000000000000000000000')
  })

  it('should have 6 decimals for pathUSD', () => {
    expect(PATH_USD_DECIMALS).toBe(6)
  })

  it('should default to testnet chain', () => {
    expect(config.chain).toBe('testnet')
  })

  it('should default to Moderato RPC URL', () => {
    expect(config.rpcUrl).toBe('https://rpc.moderato.tempo.xyz')
  })

  it('should default settlement port to 8090', () => {
    expect(config.port).toBe(8090)
  })

  it('should default coordinator URL to localhost:8080', () => {
    expect(config.coordinatorUrl).toBe('http://localhost:8080')
  })

  it('should have testnet and mainnet chain configs', () => {
    expect(TEMPO_CHAINS.testnet).toBeDefined()
    expect(TEMPO_CHAINS.mainnet).toBeDefined()
  })

  it('should have correct chain IDs', () => {
    expect(TEMPO_CHAINS.testnet.id).toBe(42429)
    expect(TEMPO_CHAINS.mainnet.id).toBe(4217)
  })
})
