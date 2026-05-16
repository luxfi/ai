#!/usr/bin/env python3
"""
Unit tests for Bitcoin tokenomics calculations
Verify all mathematical assertions in the analysis
"""

import unittest
from bitcoin_tokenomics_calculator import BitcoinTokenomics


class TestBitcoinTokenomics(unittest.TestCase):
    """Test suite for Bitcoin tokenomics calculations."""

    def setUp(self):
        """Set up test fixtures."""
        self.btc = BitcoinTokenomics()

    def test_total_supply_near_21_million(self):
        """Test that total supply is approximately 21M BTC."""
        total = self.btc.total_supply()
        self.assertAlmostEqual(total, 21_000_000, delta=1)
        print(f"✓ Total supply: {total:,.8f} BTC (within 1 BTC of 21M)")

    def test_total_halvings_is_33(self):
        """Test that there are exactly 33 halvings."""
        self.assertEqual(self.btc.total_halvings, 33)
        print(f"✓ Total halvings: {self.btc.total_halvings}")

    def test_first_halving_is_50_percent(self):
        """Test that first halving represents exactly 50% of supply."""
        supply_at_first_halving = self.btc.supply_at_block(self.btc.HALVING_INTERVAL)
        total_supply = self.btc.total_supply()
        percentage = (supply_at_first_halving / total_supply) * 100

        self.assertAlmostEqual(percentage, 50.0, delta=0.1)
        print(f"✓ First halving: {percentage:.2f}% of total supply")

    def test_50_percent_in_4_years(self):
        """Test that 50% is mined in approximately 4 years."""
        result = self.btc.time_to_percentage(50)
        self.assertAlmostEqual(result['years'], 4.0, delta=0.1)
        print(f"✓ 50% mined in: {result['years']:.2f} years")

    def test_90_percent_timeline(self):
        """Test that 90% is mined in approximately 13.6 years."""
        result = self.btc.time_to_percentage(90)
        self.assertAlmostEqual(result['years'], 13.6, delta=0.1)
        self.assertEqual(result['target_date'][:4], '2022')  # Year 2022
        print(f"✓ 90% mined in: {result['years']:.2f} years ({result['target_date']})")

    def test_99_percent_timeline(self):
        """Test that 99% is mined by ~2035."""
        result = self.btc.time_to_percentage(99)
        self.assertAlmostEqual(result['years'], 27, delta=1)
        self.assertEqual(result['target_date'][:4], '2035')
        print(f"✓ 99% mined in: {result['years']:.2f} years ({result['target_date']})")

    def test_geometric_series_formula(self):
        """Test that manual sum equals formula calculation."""
        # Manual geometric series sum
        manual_sum = sum(
            self.btc.INITIAL_REWARD * self.btc.HALVING_INTERVAL / (2 ** i)
            for i in range(self.btc.total_halvings)
        )

        # Formula calculation
        formula_result = self.btc.total_supply()

        self.assertAlmostEqual(manual_sum, formula_result, delta=0.001)
        print(f"✓ Geometric series verification: {abs(manual_sum - formula_result):.10f} BTC difference")

    def test_inflation_rate_epoch_4(self):
        """Test current inflation rate (epoch 4, 2024)."""
        inflation = self.btc.inflation_rate_at_epoch(4, use_end_supply=False)

        # Should be approximately 0.83%
        self.assertAlmostEqual(inflation, 0.83, delta=0.01)
        print(f"✓ Epoch 4 inflation rate: {inflation:.2f}%")

    def test_bitcoin_more_scarce_than_gold(self):
        """Test that Bitcoin (epoch 4) is more scarce than gold."""
        btc_inflation = self.btc.inflation_rate_at_epoch(4)
        gold_inflation = 1.5  # Annual mine production

        self.assertLess(btc_inflation, gold_inflation)
        print(f"✓ Bitcoin ({btc_inflation:.2f}%) more scarce than gold ({gold_inflation}%)")

    def test_halving_schedule_first_epoch(self):
        """Test first halving details."""
        schedule = self.btc.halving_schedule()
        first_halving = schedule[1]  # Epoch 1

        self.assertEqual(first_halving['halving'], 1)
        self.assertEqual(first_halving['block_height'], 210_000)
        self.assertEqual(first_halving['reward_btc'], 25.0)
        self.assertEqual(first_halving['year'], 2012)
        print(f"✓ First halving (Epoch 1): {first_halving['reward_btc']} BTC in {first_halving['year']}")

    def test_reward_less_than_satoshi_at_epoch_33(self):
        """Test that reward at epoch 33 is less than 1 satoshi."""
        reward_btc = self.btc.INITIAL_REWARD / (2 ** 33)
        reward_satoshis = reward_btc * self.btc.SATOSHIS_PER_BTC

        self.assertLess(reward_satoshis, 1)
        print(f"✓ Epoch 33 reward: {reward_satoshis:.10f} satoshis (< 1)")

    def test_block_time_is_10_minutes(self):
        """Test that target block time is 10 minutes."""
        self.assertEqual(self.btc.TARGET_BLOCK_TIME, 10 * 60)
        print(f"✓ Target block time: {self.btc.TARGET_BLOCK_TIME // 60} minutes")

    def test_halving_interval_is_210k(self):
        """Test that halving interval is 210,000 blocks."""
        self.assertEqual(self.btc.HALVING_INTERVAL, 210_000)
        print(f"✓ Halving interval: {self.btc.HALVING_INTERVAL:,} blocks")

    def test_cumulative_supply_increases_monotonically(self):
        """Test that supply never decreases."""
        previous_supply = 0
        for block in [0, 100_000, 210_000, 420_000, 630_000, 840_000]:
            current_supply = self.btc.supply_at_block(block)
            self.assertGreaterEqual(current_supply, previous_supply)
            previous_supply = current_supply

        print(f"✓ Supply increases monotonically: {previous_supply:,.2f} BTC at block 840,000")

    def test_inflation_decreases_each_epoch(self):
        """Test that inflation rate decreases with each halving."""
        previous_inflation = float('inf')

        for epoch in range(1, 10):
            current_inflation = self.btc.inflation_rate_at_epoch(epoch)
            self.assertLess(current_inflation, previous_inflation)
            previous_inflation = current_inflation

        print(f"✓ Inflation decreases each epoch: {current_inflation:.4f}% at epoch 9")

    def test_final_supply_convergence(self):
        """Test that supply converges to 21M as blocks approach infinity."""
        # Supply at different late stages
        supply_at_epoch_20 = self.btc.supply_at_block(20 * 210_000)
        supply_at_epoch_30 = self.btc.supply_at_block(30 * 210_000)

        total_supply = self.btc.total_supply()

        # Both should be very close to 21M
        self.assertGreater(supply_at_epoch_20, total_supply * 0.999)
        self.assertGreater(supply_at_epoch_30, total_supply * 0.99999)

        print(f"✓ Supply convergence: Epoch 20 = {supply_at_epoch_20:,.2f} BTC, "
              f"Epoch 30 = {supply_at_epoch_30:,.2f} BTC")

    def test_satoshi_precision(self):
        """Test that calculations respect satoshi precision."""
        # 1 satoshi = 0.00000001 BTC
        satoshi_in_btc = 1 / self.btc.SATOSHIS_PER_BTC
        self.assertEqual(satoshi_in_btc, 0.00000001)
        print(f"✓ Satoshi precision: {satoshi_in_btc:.8f} BTC")


def run_tests_with_output():
    """Run tests and print results."""
    print("="*80)
    print("BITCOIN TOKENOMICS CALCULATION VERIFICATION")
    print("="*80)
    print("\nRunning unit tests...\n")

    # Create test suite
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromTestCase(TestBitcoinTokenomics)

    # Run with verbose output
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    # Summary
    print("\n" + "="*80)
    print("TEST SUMMARY")
    print("="*80)
    print(f"Tests run: {result.testsRun}")
    print(f"Successes: {result.testsRun - len(result.failures) - len(result.errors)}")
    print(f"Failures: {len(result.failures)}")
    print(f"Errors: {len(result.errors)}")

    if result.wasSuccessful():
        print("\n✅ ALL TESTS PASSED - Calculations verified!")
    else:
        print("\n❌ SOME TESTS FAILED - Review errors above")

    print("="*80 + "\n")

    return result.wasSuccessful()


if __name__ == "__main__":
    import sys
    success = run_tests_with_output()
    sys.exit(0 if success else 1)
