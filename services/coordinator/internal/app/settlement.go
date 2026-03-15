package app

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	eip712DomainTypeHash      = crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	settlementVoucherTypeHash = crypto.Keccak256Hash([]byte("SettlementVoucher(address consumer,address provider,uint256 amount,uint256 platformFee,uint256 nonce,bytes32 jobId,uint256 deadline)"))
	nameHash                  = crypto.Keccak256Hash([]byte("DGInfLedger"))
	versionHash               = crypto.Keccak256Hash([]byte("1"))
	addressABIType, _         = abi.NewType("address", "", nil)
	uint256ABIType, _         = abi.NewType("uint256", "", nil)
	bytes32ABIType, _         = abi.NewType("bytes32", "", nil)
)

func makeSettlementResponse(
	key *ecdsa.PrivateKey,
	chainID uint64,
	contract string,
	job domain.JobRecord,
	consumer string,
	provider string,
	deadline time.Time,
) (domain.SettlementVoucherResponse, error) {
	if key == nil {
		return domain.SettlementVoucherResponse{}, errors.New("settlement signer not configured")
	}
	consumerAddress := common.HexToAddress(consumer)
	providerAddress := common.HexToAddress(provider)
	contractAddress := common.HexToAddress(contract)
	jobIDHash := crypto.Keccak256Hash([]byte(job.JobID))

	voucher := domain.SettlementVoucher{
		Consumer:    consumerAddress.Hex(),
		Provider:    providerAddress.Hex(),
		Amount:      job.BilledUSDC,
		PlatformFee: 0,
		Nonce:       job.SettlementNonce,
		JobIDHash:   jobIDHash.Hex(),
		Deadline:    deadline.Unix(),
	}

	digest, err := settlementDigest(chainID, contractAddress, voucher)
	if err != nil {
		return domain.SettlementVoucherResponse{}, err
	}
	signature, err := crypto.Sign(digest.Bytes(), key)
	if err != nil {
		return domain.SettlementVoucherResponse{}, err
	}
	signature[64] += 27
	return domain.SettlementVoucherResponse{
		Voucher:        voucher,
		Signature:      "0x" + hex.EncodeToString(signature),
		SignerAddress:  crypto.PubkeyToAddress(key.PublicKey).Hex(),
		VerifyingChain: chainID,
		Contract:       contractAddress.Hex(),
	}, nil
}

func settlementDigest(chainID uint64, contract common.Address, voucher domain.SettlementVoucher) (common.Hash, error) {
	args := abi.Arguments{
		{Type: bytes32ABIType},
		{Type: bytes32ABIType},
		{Type: bytes32ABIType},
		{Type: uint256ABIType},
		{Type: addressABIType},
	}
	domainEncoded, err := args.Pack(
		eip712DomainTypeHash,
		nameHash,
		versionHash,
		new(big.Int).SetUint64(chainID),
		contract,
	)
	if err != nil {
		return common.Hash{}, err
	}
	domainSeparator := crypto.Keccak256Hash(domainEncoded)

	structArgs := abi.Arguments{
		{Type: bytes32ABIType},
		{Type: addressABIType},
		{Type: addressABIType},
		{Type: uint256ABIType},
		{Type: uint256ABIType},
		{Type: uint256ABIType},
		{Type: bytes32ABIType},
		{Type: uint256ABIType},
	}
	jobIDHash := common.HexToHash(strings.TrimPrefix(voucher.JobIDHash, "0x"))
	structEncoded, err := structArgs.Pack(
		settlementVoucherTypeHash,
		common.HexToAddress(voucher.Consumer),
		common.HexToAddress(voucher.Provider),
		big.NewInt(voucher.Amount),
		big.NewInt(voucher.PlatformFee),
		new(big.Int).SetUint64(voucher.Nonce),
		jobIDHash,
		big.NewInt(voucher.Deadline),
	)
	if err != nil {
		return common.Hash{}, err
	}
	structHash := crypto.Keccak256Hash(structEncoded)
	raw := append([]byte{0x19, 0x01}, domainSeparator.Bytes()...)
	raw = append(raw, structHash.Bytes()...)
	return crypto.Keccak256Hash(raw), nil
}
