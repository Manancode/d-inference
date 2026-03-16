import { tempoTestnet, tempo } from 'viem/chains';
export const TEMPO_CHAINS = {
    testnet: tempoTestnet,
    mainnet: tempo,
};
export const PATH_USD = '0x20C0000000000000000000000000000000000000';
export const PATH_USD_DECIMALS = 6;
// TIP-20 ABI (subset we need)
export const TIP20_ABI = [
    {
        name: 'transfer',
        type: 'function',
        inputs: [
            { name: 'to', type: 'address' },
            { name: 'amount', type: 'uint256' },
        ],
        outputs: [{ type: 'bool' }],
        stateMutability: 'nonpayable',
    },
    {
        name: 'transferWithMemo',
        type: 'function',
        inputs: [
            { name: 'to', type: 'address' },
            { name: 'amount', type: 'uint256' },
            { name: 'memo', type: 'bytes32' },
        ],
        outputs: [],
        stateMutability: 'nonpayable',
    },
    {
        name: 'balanceOf',
        type: 'function',
        inputs: [{ name: 'account', type: 'address' }],
        outputs: [{ type: 'uint256' }],
        stateMutability: 'view',
    },
    {
        name: 'Transfer',
        type: 'event',
        inputs: [
            { name: 'from', type: 'address', indexed: true },
            { name: 'to', type: 'address', indexed: true },
            { name: 'value', type: 'uint256', indexed: false },
        ],
    },
];
// DGInf platform wallet (receives deposits, sends payouts).
// In production this would be loaded from env/secrets.
export const PLATFORM_WALLET = process.env.DGINF_PLATFORM_WALLET || '0x0000000000000000000000000000000000000000';
export const config = {
    chain: process.env.DGINF_CHAIN === 'mainnet' ? 'mainnet' : 'testnet',
    rpcUrl: process.env.DGINF_RPC_URL || 'https://rpc.moderato.tempo.xyz',
    platformWallet: PLATFORM_WALLET,
    coordinatorUrl: process.env.DGINF_COORDINATOR_URL || 'http://localhost:8080',
    port: parseInt(process.env.DGINF_SETTLEMENT_PORT || '8090'),
};
//# sourceMappingURL=config.js.map