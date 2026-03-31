package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// EVMProcessor handles deposit verification and withdrawals on EVM-compatible chains.
//
// Supported chains: Ethereum, Tempo (pathUSD), Base, or any EVM chain.
//
// Deposit flow:
//  1. Consumer sends USDC/pathUSD to the deposit address
//  2. Consumer submits tx hash to coordinator
//  3. We verify the tx via JSON-RPC (eth_getTransactionReceipt)
//  4. We parse Transfer event logs to confirm amount and recipient
//  5. Credits consumer's internal balance
//
// Withdrawal flow:
//  1. Consumer requests withdrawal with destination address
//  2. Coordinator signs and sends ERC-20 transfer from hot wallet
//  3. Returns tx hash to consumer
type EVMProcessor struct {
	config     EVMChainConfig
	logger     *slog.Logger
	httpClient *http.Client
}

// NewEVMProcessor creates a new EVM chain processor.
func NewEVMProcessor(config EVMChainConfig, logger *slog.Logger) *EVMProcessor {
	return &EVMProcessor{
		config:     config,
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DepositAddress returns the address consumers should send tokens to.
func (p *EVMProcessor) DepositAddress() string {
	return p.config.DepositAddress
}

// Chain returns which chain this processor handles.
func (p *EVMProcessor) Chain() Chain {
	return p.config.Chain
}

// USDCContract returns the stablecoin contract address for this chain.
func (p *EVMProcessor) USDCContract() string {
	return p.config.USDCContract
}

// EVMDepositResult contains the verified deposit details.
type EVMDepositResult struct {
	TxHash         string `json:"tx_hash"`
	From           string `json:"from"`
	To             string `json:"to"`
	AmountRaw      string `json:"amount_raw"`       // raw token amount (before decimals)
	AmountMicroUSD int64  `json:"amount_micro_usd"` // converted to micro-USD (6 decimals)
	BlockNumber    int64  `json:"block_number"`
	Confirmed      bool   `json:"confirmed"`
}

// ERC-20 Transfer event topic: keccak256("Transfer(address,address,uint256)")
const erc20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

// VerifyDeposit verifies an on-chain ERC-20 transfer to the deposit address.
func (p *EVMProcessor) VerifyDeposit(txHash string) (*EVMDepositResult, error) {
	// Get transaction receipt via JSON-RPC
	receipt, err := p.getTransactionReceipt(txHash)
	if err != nil {
		return nil, fmt.Errorf("evm: get receipt: %w", err)
	}

	// Verify transaction succeeded
	status, ok := receipt["status"].(string)
	if !ok || status != "0x1" {
		return nil, fmt.Errorf("evm: transaction failed or pending (status: %v)", receipt["status"])
	}

	// Parse block number
	blockNumHex, _ := receipt["blockNumber"].(string)
	blockNum := hexToInt64(blockNumHex)

	// Check confirmation depth (require at least 1 confirmation)
	currentBlock, err := p.getBlockNumber()
	if err != nil {
		return nil, fmt.Errorf("evm: get current block: %w", err)
	}
	if currentBlock-blockNum < 1 {
		return nil, fmt.Errorf("evm: transaction not yet confirmed (block %d, current %d)", blockNum, currentBlock)
	}

	// Parse Transfer event logs
	logs, ok := receipt["logs"].([]any)
	if !ok {
		return nil, fmt.Errorf("evm: no logs in transaction receipt")
	}

	depositAddr := strings.ToLower(p.config.DepositAddress)
	usdcContract := strings.ToLower(p.config.USDCContract)

	for _, logEntry := range logs {
		logMap, ok := logEntry.(map[string]any)
		if !ok {
			continue
		}

		// Check if this log is from the USDC contract
		addr, _ := logMap["address"].(string)
		if strings.ToLower(addr) != usdcContract {
			continue
		}

		// Check if this is a Transfer event
		topics, ok := logMap["topics"].([]any)
		if !ok || len(topics) < 3 {
			continue
		}

		topic0, _ := topics[0].(string)
		if strings.ToLower(topic0) != erc20TransferTopic {
			continue
		}

		// Parse from and to addresses from topics (32-byte padded)
		fromTopic, _ := topics[1].(string)
		toTopic, _ := topics[2].(string)
		from := topicToAddress(fromTopic)
		to := topicToAddress(toTopic)

		// Verify the transfer is to our deposit address
		if strings.ToLower(to) != depositAddr {
			continue
		}

		// Parse amount from data field
		data, _ := logMap["data"].(string)
		amountRaw := hexToBigInt(data)

		// USDC and pathUSD both use 6 decimals, which maps 1:1 to micro-USD
		amountMicroUSD := amountRaw.Int64()

		return &EVMDepositResult{
			TxHash:         txHash,
			From:           from,
			To:             to,
			AmountRaw:      amountRaw.String(),
			AmountMicroUSD: amountMicroUSD,
			BlockNumber:    blockNum,
			Confirmed:      true,
		}, nil
	}

	return nil, fmt.Errorf("evm: no matching USDC/pathUSD Transfer event to deposit address in tx %s", txHash)
}

// rpcCall makes a JSON-RPC call to the EVM node.
func (p *EVMProcessor) rpcCall(method string, params []any) (json.RawMessage, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Post(p.config.RPCURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc response: %w", err)
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// getTransactionReceipt fetches the receipt for a transaction.
func (p *EVMProcessor) getTransactionReceipt(txHash string) (map[string]any, error) {
	result, err := p.rpcCall("eth_getTransactionReceipt", []any{txHash})
	if err != nil {
		return nil, err
	}

	if string(result) == "null" {
		return nil, fmt.Errorf("transaction not found or not yet mined")
	}

	var receipt map[string]any
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, fmt.Errorf("parse receipt: %w", err)
	}
	return receipt, nil
}

// getBlockNumber returns the current block number.
func (p *EVMProcessor) getBlockNumber() (int64, error) {
	result, err := p.rpcCall("eth_blockNumber", []any{})
	if err != nil {
		return 0, err
	}

	var blockHex string
	if err := json.Unmarshal(result, &blockHex); err != nil {
		return 0, err
	}
	return hexToInt64(blockHex), nil
}

// topicToAddress extracts an Ethereum address from a 32-byte log topic.
// Topics are zero-padded to 32 bytes, so "0x000...000<address>" → "0x<address>".
func topicToAddress(topic string) string {
	topic = strings.TrimPrefix(topic, "0x")
	if len(topic) < 40 {
		return "0x" + topic
	}
	return "0x" + topic[len(topic)-40:]
}

// hexToInt64 parses a 0x-prefixed hex string to int64.
func hexToInt64(hex string) int64 {
	hex = strings.TrimPrefix(hex, "0x")
	n := new(big.Int)
	n.SetString(hex, 16)
	return n.Int64()
}

// hexToBigInt parses a 0x-prefixed hex string to *big.Int.
func hexToBigInt(hex string) *big.Int {
	hex = strings.TrimPrefix(hex, "0x")
	n := new(big.Int)
	n.SetString(hex, 16)
	return n
}

// SendWithdrawal sends an ERC-20 transfer from the hot wallet.
// This is a placeholder — production implementation would use a proper
// transaction signing library (e.g., go-ethereum's ethclient).
type EVMWithdrawRequest struct {
	ToAddress      string `json:"to_address"`
	AmountMicroUSD int64  `json:"amount_micro_usd"`
}

type EVMWithdrawResult struct {
	TxHash         string `json:"tx_hash"`
	ToAddress      string `json:"to_address"`
	AmountMicroUSD int64  `json:"amount_micro_usd"`
	Chain          Chain  `json:"chain"`
}

// SendWithdrawal initiates an on-chain withdrawal. In production this would
// construct and sign an ERC-20 transfer transaction. For now, it calls the
// settlement sidecar service if configured.
func (p *EVMProcessor) SendWithdrawal(req EVMWithdrawRequest, settlementURL string) (*EVMWithdrawResult, error) {
	if settlementURL == "" {
		return nil, fmt.Errorf("evm: settlement service not configured for withdrawals")
	}

	withdrawBody, err := json.Marshal(map[string]any{
		"to_address":       req.ToAddress,
		"amount_micro_usd": req.AmountMicroUSD,
		"chain":            string(p.config.Chain),
		"reason":           "consumer_withdrawal",
	})
	if err != nil {
		return nil, fmt.Errorf("evm: marshal withdraw request: %w", err)
	}

	resp, err := p.httpClient.Post(
		settlementURL+"/v1/settlement/withdraw",
		"application/json",
		bytes.NewReader(withdrawBody),
	)
	if err != nil {
		return nil, fmt.Errorf("evm: settlement service unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("evm: read settlement response: %w", err)
	}

	var result struct {
		TxHash  string `json:"txHash"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("evm: parse settlement response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("evm: withdrawal failed: %s", result.Error)
	}

	return &EVMWithdrawResult{
		TxHash:         result.TxHash,
		ToAddress:      req.ToAddress,
		AmountMicroUSD: req.AmountMicroUSD,
		Chain:          p.config.Chain,
	}, nil
}
