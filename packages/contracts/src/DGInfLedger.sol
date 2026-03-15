// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

contract DGInfLedger {
    error Unauthorized();
    error InvalidAddress();
    error InvalidBasisPoints();
    error InvalidAmount();
    error InvalidNonce();
    error VoucherExpired();
    error InvalidSignature();
    error InsufficientBalance();
    error TransferFailed();

    struct SettlementVoucher {
        address consumer;
        address provider;
        uint256 amount;
        uint256 platformFee;
        uint256 nonce;
        bytes32 jobId;
        uint256 deadline;
    }

    bytes32 public constant SETTLEMENT_VOUCHER_TYPEHASH =
        keccak256(
            "SettlementVoucher(address consumer,address provider,uint256 amount,uint256 platformFee,uint256 nonce,bytes32 jobId,uint256 deadline)"
        );

    IERC20 public immutable usdc;
    address public owner;
    address public coordinatorSigner;
    address public treasuryWallet;
    uint16 public platformFeeBps;

    mapping(address => uint256) public consumerBalance;
    mapping(address => uint256) public providerBalance;
    mapping(address => uint256) public treasuryBalance;
    mapping(address => uint256) public settlementNonce;

    event ConsumerDeposited(address indexed consumer, uint256 amount);
    event ConsumerWithdrawn(address indexed consumer, uint256 amount);
    event ProviderWithdrawn(address indexed provider, uint256 amount);
    event TreasuryWithdrawn(address indexed treasury, uint256 amount);
    event SettlementApplied(
        address indexed consumer,
        address indexed provider,
        bytes32 indexed jobId,
        uint256 amount,
        uint256 platformFee,
        uint256 nonce
    );
    event CoordinatorSignerUpdated(address indexed signer);
    event TreasuryWalletUpdated(address indexed treasuryWallet);
    event PlatformFeeUpdated(uint16 platformFeeBps);

    modifier onlyOwner() {
        if (msg.sender != owner) revert Unauthorized();
        _;
    }

    modifier onlyTreasury() {
        if (msg.sender != treasuryWallet) revert Unauthorized();
        _;
    }

    constructor(address usdc_, address coordinatorSigner_, address treasuryWallet_) {
        if (usdc_ == address(0) || coordinatorSigner_ == address(0) || treasuryWallet_ == address(0)) {
            revert InvalidAddress();
        }

        usdc = IERC20(usdc_);
        owner = msg.sender;
        coordinatorSigner = coordinatorSigner_;
        treasuryWallet = treasuryWallet_;
    }

    function depositConsumer(uint256 amount) external {
        if (amount == 0) revert InvalidAmount();
        _safeTransferFrom(msg.sender, address(this), amount);
        consumerBalance[msg.sender] += amount;
        emit ConsumerDeposited(msg.sender, amount);
    }

    function withdrawConsumer(uint256 amount) external {
        if (amount == 0) revert InvalidAmount();
        if (consumerBalance[msg.sender] < amount) revert InsufficientBalance();

        consumerBalance[msg.sender] -= amount;
        _safeTransfer(msg.sender, amount);
        emit ConsumerWithdrawn(msg.sender, amount);
    }

    function withdrawProvider(uint256 amount) external {
        if (amount == 0) revert InvalidAmount();
        if (providerBalance[msg.sender] < amount) revert InsufficientBalance();

        providerBalance[msg.sender] -= amount;
        _safeTransfer(msg.sender, amount);
        emit ProviderWithdrawn(msg.sender, amount);
    }

    function withdrawTreasury(uint256 amount) external onlyTreasury {
        if (amount == 0) revert InvalidAmount();
        if (treasuryBalance[msg.sender] < amount) revert InsufficientBalance();

        treasuryBalance[msg.sender] -= amount;
        _safeTransfer(msg.sender, amount);
        emit TreasuryWithdrawn(msg.sender, amount);
    }

    function settle(SettlementVoucher calldata voucher, bytes calldata signature) external {
        if (block.timestamp > voucher.deadline) revert VoucherExpired();
        if (voucher.consumer == address(0) || voucher.provider == address(0)) revert InvalidAddress();
        if (voucher.amount == 0) revert InvalidAmount();
        if (voucher.platformFee > voucher.amount) revert InvalidAmount();
        if (voucher.nonce != settlementNonce[voucher.consumer] + 1) revert InvalidNonce();

        uint256 totalDebit = voucher.amount + voucher.platformFee;
        if (consumerBalance[voucher.consumer] < totalDebit) revert InsufficientBalance();
        if (!_isValidSignature(voucher, signature)) revert InvalidSignature();

        settlementNonce[voucher.consumer] = voucher.nonce;
        consumerBalance[voucher.consumer] -= totalDebit;
        providerBalance[voucher.provider] += voucher.amount;
        treasuryBalance[treasuryWallet] += voucher.platformFee;

        emit SettlementApplied(
            voucher.consumer,
            voucher.provider,
            voucher.jobId,
            voucher.amount,
            voucher.platformFee,
            voucher.nonce
        );
    }

    function setCoordinatorSigner(address signer) external onlyOwner {
        if (signer == address(0)) revert InvalidAddress();
        coordinatorSigner = signer;
        emit CoordinatorSignerUpdated(signer);
    }

    function setTreasuryWallet(address treasuryWallet_) external onlyOwner {
        if (treasuryWallet_ == address(0)) revert InvalidAddress();
        treasuryWallet = treasuryWallet_;
        emit TreasuryWalletUpdated(treasuryWallet_);
    }

    function setPlatformFeeBps(uint16 bps) external onlyOwner {
        if (bps > 10_000) revert InvalidBasisPoints();
        platformFeeBps = bps;
        emit PlatformFeeUpdated(bps);
    }

    function domainSeparator() public view returns (bytes32) {
        return keccak256(
            abi.encode(
                keccak256(
                    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
                ),
                keccak256(bytes("DGInfLedger")),
                keccak256(bytes("1")),
                block.chainid,
                address(this)
            )
        );
    }

    function digest(SettlementVoucher calldata voucher) public view returns (bytes32) {
        bytes32 structHash = keccak256(
            abi.encode(
                SETTLEMENT_VOUCHER_TYPEHASH,
                voucher.consumer,
                voucher.provider,
                voucher.amount,
                voucher.platformFee,
                voucher.nonce,
                voucher.jobId,
                voucher.deadline
            )
        );

        return keccak256(abi.encodePacked("\x19\x01", domainSeparator(), structHash));
    }

    function _isValidSignature(SettlementVoucher calldata voucher, bytes calldata signature)
        internal
        view
        returns (bool)
    {
        if (signature.length != 65) return false;

        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := calldataload(signature.offset)
            s := calldataload(add(signature.offset, 32))
            v := byte(0, calldataload(add(signature.offset, 64)))
        }

        return ecrecover(digest(voucher), v, r, s) == coordinatorSigner;
    }

    function _safeTransfer(address to, uint256 amount) internal {
        if (!usdc.transfer(to, amount)) revert TransferFailed();
    }

    function _safeTransferFrom(address from, address to, uint256 amount) internal {
        if (!usdc.transferFrom(from, to, amount)) revert TransferFailed();
    }
}

