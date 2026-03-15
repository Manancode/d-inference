// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {DGInfLedger} from "../src/DGInfLedger.sol";
import {MockUSDC} from "./MockUSDC.sol";

interface Vm {
    function addr(uint256 privateKey) external returns (address);
    function prank(address caller) external;
    function expectRevert(bytes4 revertData) external;
    function sign(uint256 privateKey, bytes32 digest) external returns (uint8 v, bytes32 r, bytes32 s);
    function assume(bool) external;
}

contract DGInfLedgerTest {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    uint256 internal constant COORDINATOR_PK = 0xA11CE;

    MockUSDC internal usdc;
    DGInfLedger internal ledger;
    address internal coordinator;
    address internal treasury = address(0xBEEF);
    address internal consumer = address(0xCAFE);
    address internal provider = address(0xD00D);

    function setUp() public {
        coordinator = vm.addr(COORDINATOR_PK);
        usdc = new MockUSDC();
        ledger = new DGInfLedger(address(usdc), coordinator, treasury);

        usdc.mint(consumer, 1_000_000_000);
        vm.prank(consumer);
        usdc.approve(address(ledger), type(uint256).max);
    }

    function testDepositAndWithdrawConsumer() public {
        vm.prank(consumer);
        ledger.depositConsumer(250_000_000);
        assert(ledger.consumerBalance(consumer) == 250_000_000);

        vm.prank(consumer);
        ledger.withdrawConsumer(100_000_000);

        assert(ledger.consumerBalance(consumer) == 150_000_000);
        assert(usdc.balanceOf(consumer) == 850_000_000);
    }

    function testSettleCreditsProviderAndTreasury() public {
        vm.prank(consumer);
        ledger.depositConsumer(400_000_000);

        DGInfLedger.SettlementVoucher memory voucher = DGInfLedger.SettlementVoucher({
            consumer: consumer,
            provider: provider,
            amount: 125_000_000,
            platformFee: 5_000_000,
            nonce: 1,
            jobId: keccak256("job-1"),
            deadline: block.timestamp + 1 days
        });

        bytes memory signature = _sign(voucher);
        ledger.settle(voucher, signature);

        assert(ledger.consumerBalance(consumer) == 270_000_000);
        assert(ledger.providerBalance(provider) == 125_000_000);
        assert(ledger.treasuryBalance(treasury) == 5_000_000);
        assert(ledger.settlementNonce(consumer) == 1);
    }

    function testSettleRejectsReplay() public {
        vm.prank(consumer);
        ledger.depositConsumer(400_000_000);

        DGInfLedger.SettlementVoucher memory voucher = DGInfLedger.SettlementVoucher({
            consumer: consumer,
            provider: provider,
            amount: 100_000_000,
            platformFee: 0,
            nonce: 1,
            jobId: keccak256("job-1"),
            deadline: block.timestamp + 1 days
        });

        bytes memory signature = _sign(voucher);
        ledger.settle(voucher, signature);

        vm.expectRevert(DGInfLedger.InvalidNonce.selector);
        ledger.settle(voucher, signature);
    }

    function testSettleRejectsOutOfOrderNonce() public {
        vm.prank(consumer);
        ledger.depositConsumer(400_000_000);

        DGInfLedger.SettlementVoucher memory voucher = DGInfLedger.SettlementVoucher({
            consumer: consumer,
            provider: provider,
            amount: 100_000_000,
            platformFee: 0,
            nonce: 2,
            jobId: keccak256("job-1"),
            deadline: block.timestamp + 1 days
        });

        bytes memory signature = _sign(voucher);
        vm.expectRevert(DGInfLedger.InvalidNonce.selector);
        ledger.settle(voucher, signature);
    }

    function testSettleRejectsInvalidSignature() public {
        vm.prank(consumer);
        ledger.depositConsumer(400_000_000);

        DGInfLedger.SettlementVoucher memory voucher = DGInfLedger.SettlementVoucher({
            consumer: consumer,
            provider: provider,
            amount: 100_000_000,
            platformFee: 0,
            nonce: 1,
            jobId: keccak256("job-1"),
            deadline: block.timestamp + 1 days
        });

        bytes memory signature = new bytes(65);
        vm.expectRevert(DGInfLedger.InvalidSignature.selector);
        ledger.settle(voucher, signature);
    }

    function testWithdrawProvider() public {
        vm.prank(consumer);
        ledger.depositConsumer(400_000_000);

        DGInfLedger.SettlementVoucher memory voucher = DGInfLedger.SettlementVoucher({
            consumer: consumer,
            provider: provider,
            amount: 175_000_000,
            platformFee: 0,
            nonce: 1,
            jobId: keccak256("job-1"),
            deadline: block.timestamp + 1 days
        });

        ledger.settle(voucher, _sign(voucher));

        vm.prank(provider);
        ledger.withdrawProvider(100_000_000);

        assert(ledger.providerBalance(provider) == 75_000_000);
        assert(usdc.balanceOf(provider) == 100_000_000);
    }

    function testFuzzDepositWithdrawRoundTrip(uint96 amount) public {
        vm.assume(amount > 0);
        uint256 bounded = uint256(amount) % 100_000_000 + 1;

        vm.prank(consumer);
        ledger.depositConsumer(bounded);
        vm.prank(consumer);
        ledger.withdrawConsumer(bounded);

        assert(ledger.consumerBalance(consumer) == 0);
        assert(usdc.balanceOf(address(ledger)) == 0);
    }

    function _sign(DGInfLedger.SettlementVoucher memory voucher) internal returns (bytes memory) {
        bytes32 digest = ledger.digest(voucher);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(COORDINATOR_PK, digest);
        return abi.encodePacked(r, s, v);
    }
}
