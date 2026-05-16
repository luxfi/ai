#!/usr/bin/env python3
"""
Bitcoin Tokenomics Calculator
Rigorous mathematical verification of Bitcoin's supply schedule and inflation rates
"""

import math
from datetime import datetime, timedelta
from typing import List, Tuple, Dict
import json


class BitcoinTokenomics:
    """
    Precise Bitcoin tokenomics calculator based on protocol specifications.
    """

    # Protocol constants
    INITIAL_REWARD = 50.0  # BTC per block
    HALVING_INTERVAL = 210_000  # blocks
    TARGET_BLOCK_TIME = 10 * 60  # seconds (10 minutes)
    SATOSHIS_PER_BTC = 100_000_000
    GENESIS_DATE = datetime(2009, 1, 3)  # Genesis block date

    def __init__(self):
        self.total_halvings = self._calculate_total_halvings()

    def _calculate_total_halvings(self) -> int:
        """
        Calculate the total number of halvings until reward < 1 satoshi.

        Returns:
            Number of halvings
        """
        reward_satoshis = self.INITIAL_REWARD * self.SATOSHIS_PER_BTC
        halvings = 0

        while reward_satoshis >= 1:
            reward_satoshis /= 2
            halvings += 1

        return halvings

    def total_supply(self) -> float:
        """
        Calculate total Bitcoin supply using geometric series.

        Returns:
            Total supply in BTC
        """
        # Sum of geometric series: a * (1 - r^n) / (1 - r)
        # where a = initial term, r = ratio (0.5), n = number of terms
        supply = self.INITIAL_REWARD * self.HALVING_INTERVAL * (1 - (0.5 ** self.total_halvings)) / (1 - 0.5)
        return supply

    def supply_at_block(self, block_height: int) -> float:
        """
        Calculate cumulative supply at a given block height.

        Args:
            block_height: Block number

        Returns:
            Cumulative supply in BTC
        """
        supply = 0.0
        blocks_processed = 0
        epoch = 0

        while blocks_processed < block_height:
            blocks_in_epoch = min(self.HALVING_INTERVAL, block_height - blocks_processed)
            reward = self.INITIAL_REWARD / (2 ** epoch)
            supply += blocks_in_epoch * reward
            blocks_processed += blocks_in_epoch
            epoch += 1

        return supply

    def blocks_to_percentage(self, percentage: float) -> Tuple[int, float]:
        """
        Calculate blocks needed to mine a given percentage of total supply.

        Args:
            percentage: Target percentage (0-100)

        Returns:
            Tuple of (blocks, exact_percentage)
        """
        target_supply = self.total_supply() * (percentage / 100)

        blocks = 0
        cumulative_supply = 0.0
        epoch = 0

        while cumulative_supply < target_supply:
            reward = self.INITIAL_REWARD / (2 ** epoch)
            remaining_supply = target_supply - cumulative_supply

            if remaining_supply >= self.HALVING_INTERVAL * reward:
                # Full epoch
                blocks += self.HALVING_INTERVAL
                cumulative_supply += self.HALVING_INTERVAL * reward
                epoch += 1
            else:
                # Partial epoch
                blocks_needed = int(remaining_supply / reward)
                blocks += blocks_needed
                cumulative_supply += blocks_needed * reward
                break

        exact_percentage = (cumulative_supply / self.total_supply()) * 100
        return blocks, exact_percentage

    def time_to_percentage(self, percentage: float) -> Dict[str, any]:
        """
        Calculate time to mine a given percentage of supply.

        Args:
            percentage: Target percentage (0-100)

        Returns:
            Dictionary with blocks, time, and date
        """
        blocks, exact_pct = self.blocks_to_percentage(percentage)

        total_seconds = blocks * self.TARGET_BLOCK_TIME
        time_delta = timedelta(seconds=total_seconds)
        target_date = self.GENESIS_DATE + time_delta

        years = time_delta.days / 365.25

        return {
            'target_percentage': percentage,
            'exact_percentage': exact_pct,
            'blocks': blocks,
            'seconds': total_seconds,
            'days': time_delta.days,
            'years': years,
            'target_date': target_date.strftime('%Y-%m-%d'),
            'supply_btc': self.supply_at_block(blocks)
        }

    def halving_schedule(self) -> List[Dict[str, any]]:
        """
        Generate complete halving schedule.

        Returns:
            List of halving events with details
        """
        schedule = []

        for epoch in range(self.total_halvings + 1):
            block_height = epoch * self.HALVING_INTERVAL
            reward = self.INITIAL_REWARD / (2 ** epoch)
            cumulative_supply = self.supply_at_block(block_height)

            # Calculate inflation rate at start of epoch
            if cumulative_supply > 0:
                annual_issuance = (365.25 * 24 * 60 / 10) * reward
                inflation_rate = (annual_issuance / cumulative_supply) * 100
            else:
                inflation_rate = float('inf')

            # Calculate date
            total_seconds = block_height * self.TARGET_BLOCK_TIME
            date = self.GENESIS_DATE + timedelta(seconds=total_seconds)
            years_from_genesis = total_seconds / (365.25 * 24 * 60 * 60)

            schedule.append({
                'halving': epoch,
                'year': date.year,
                'date': date.strftime('%Y-%m-%d'),
                'years_from_genesis': round(years_from_genesis, 2),
                'block_height': block_height,
                'reward_btc': reward,
                'reward_satoshis': reward * self.SATOSHIS_PER_BTC,
                'annual_issuance_btc': round((365.25 * 24 * 60 / 10) * reward, 2),
                'cumulative_supply_btc': round(cumulative_supply, 2),
                'supply_percentage': round((cumulative_supply / self.total_supply()) * 100, 4),
                'inflation_rate_percent': round(inflation_rate, 4) if inflation_rate != float('inf') else 'N/A'
            })

        return schedule

    def inflation_rate_at_epoch(self, epoch: int, use_end_supply: bool = False) -> float:
        """
        Calculate inflation rate for a given epoch.

        Args:
            epoch: Halving epoch number
            use_end_supply: If True, use supply at end of epoch; otherwise use start

        Returns:
            Annual inflation rate as percentage
        """
        if use_end_supply:
            block_height = (epoch + 1) * self.HALVING_INTERVAL
        else:
            block_height = epoch * self.HALVING_INTERVAL

        existing_supply = self.supply_at_block(block_height)
        reward = self.INITIAL_REWARD / (2 ** epoch)
        annual_issuance = (365.25 * 24 * 60 / 10) * reward

        if existing_supply == 0:
            return float('inf')

        return (annual_issuance / existing_supply) * 100

    def comparative_analysis(self) -> Dict[str, any]:
        """
        Comparative analysis with other assets/currencies.

        Returns:
            Dictionary with comparisons
        """
        current_epoch = 4  # As of 2024
        btc_inflation = self.inflation_rate_at_epoch(current_epoch)

        return {
            'bitcoin_current_inflation': round(btc_inflation, 2),
            'gold_inflation': 1.5,  # Annual mine production ~1.5%
            'us_dollar_m2_avg': 7.0,  # Historical average
            'target_inflation_fed': 2.0,
            'bitcoin_vs_gold': 'More scarce' if btc_inflation < 1.5 else 'Less scarce',
            'bitcoin_vs_usd': 'More scarce' if btc_inflation < 2.0 else 'Less scarce'
        }


def print_section(title: str):
    """Pretty print section headers."""
    print(f"\n{'='*80}")
    print(f"{title.center(80)}")
    print(f"{'='*80}\n")


def main():
    """
    Main execution: Calculate and display all Bitcoin tokenomics parameters.
    """
    btc = BitcoinTokenomics()

    # 1. Core Parameters
    print_section("BITCOIN TOKENOMICS ANALYSIS")

    print(f"Total Supply (calculated): {btc.total_supply():,.8f} BTC")
    print(f"Total Supply (theoretical): 21,000,000.00000000 BTC")
    print(f"Difference: {abs(21_000_000 - btc.total_supply()):,.8f} BTC")
    print(f"Total Halvings: {btc.total_halvings}")
    print(f"Initial Block Reward: {btc.INITIAL_REWARD} BTC")
    print(f"Halving Interval: {btc.HALVING_INTERVAL:,} blocks")
    print(f"Target Block Time: {btc.TARGET_BLOCK_TIME // 60} minutes")

    # 2. Supply Distribution Milestones
    print_section("SUPPLY DISTRIBUTION TIMELINE")

    milestones = [50, 75, 90, 95, 99, 99.9, 99.99]

    for pct in milestones:
        result = btc.time_to_percentage(pct)
        print(f"\n{pct}% of supply ({result['supply_btc']:,.2f} BTC):")
        print(f"  Blocks needed: {result['blocks']:,}")
        print(f"  Time: {result['years']:.2f} years ({result['days']:,} days)")
        print(f"  Target date: {result['target_date']}")
        print(f"  Exact percentage: {result['exact_percentage']:.6f}%")

    # 3. Halving Schedule
    print_section("HALVING SCHEDULE (First 10 + Last 5)")

    schedule = btc.halving_schedule()

    # Print header
    print(f"{'Epoch':<6} {'Year':<6} {'Block Height':<15} {'Reward (BTC)':<15} "
          f"{'Annual Issuance':<18} {'Supply %':<12} {'Inflation %':<15}")
    print("-" * 110)

    # First 10 halvings
    for event in schedule[:10]:
        print(f"{event['halving']:<6} {event['year']:<6} {event['block_height']:<15,} "
              f"{event['reward_btc']:<15.8f} {event['annual_issuance_btc']:<18,} "
              f"{event['supply_percentage']:<12.4f} {str(event['inflation_rate_percent']):<15}")

    print("...")

    # Last 5 halvings
    for event in schedule[-5:]:
        print(f"{event['halving']:<6} {event['year']:<6} {event['block_height']:<15,} "
              f"{event['reward_btc']:<15.10f} {event['annual_issuance_btc']:<18,} "
              f"{event['supply_percentage']:<12.4f} {str(event['inflation_rate_percent']):<15}")

    # 4. Inflation Rates by Epoch
    print_section("INFLATION RATES BY EPOCH (Detailed)")

    print(f"{'Epoch':<8} {'Period':<15} {'Start Inflation %':<20} {'End Inflation %':<20}")
    print("-" * 70)

    for epoch in range(min(10, btc.total_halvings)):
        start_year = 2009 + (epoch * 4)
        end_year = start_year + 4
        start_inflation = btc.inflation_rate_at_epoch(epoch, use_end_supply=False)
        end_inflation = btc.inflation_rate_at_epoch(epoch, use_end_supply=True)

        start_str = f"{start_inflation:.4f}" if start_inflation != float('inf') else "∞"
        end_str = f"{end_inflation:.4f}" if end_inflation != float('inf') else "∞"

        print(f"{epoch:<8} {start_year}-{end_year:<8} {start_str:<20} {end_str:<20}")

    # 5. Comparative Analysis
    print_section("COMPARATIVE ANALYSIS (2024)")

    comparison = btc.comparative_analysis()

    print(f"Bitcoin Current Inflation: {comparison['bitcoin_current_inflation']:.2f}%")
    print(f"Gold Annual Production: {comparison['gold_inflation']:.2f}%")
    print(f"US Dollar M2 (Historical Avg): {comparison['us_dollar_m2_avg']:.2f}%")
    print(f"Federal Reserve Target: {comparison['target_inflation_fed']:.2f}%")
    print(f"\nBitcoin vs. Gold: {comparison['bitcoin_vs_gold']}")
    print(f"Bitcoin vs. USD: {comparison['bitcoin_vs_usd']}")

    # 6. Security Budget Analysis
    print_section("SECURITY BUDGET PROJECTION")

    print(f"{'Epoch':<8} {'Year':<8} {'Block Reward':<15} {'Annual Issuance':<18} "
          f"{'Value @ $100k':<20}")
    print("-" * 75)

    btc_price = 100_000  # Assume $100k BTC

    for epoch in [4, 5, 6, 7, 8, 10, 15, 20]:
        year = 2009 + (epoch * 4)
        reward = btc.INITIAL_REWARD / (2 ** epoch)
        annual_issuance = (365.25 * 24 * 60 / 10) * reward
        value_usd = annual_issuance * btc_price

        print(f"{epoch:<8} {year:<8} {reward:<15.8f} {annual_issuance:<18,.2f} "
              f"${value_usd:<19,.0f}")

    print("\nNote: Security budget must transition to transaction fees as block rewards diminish.")

    # 7. Export to JSON
    print_section("EXPORTING DATA")

    output_data = {
        'metadata': {
            'total_supply_btc': btc.total_supply(),
            'total_halvings': btc.total_halvings,
            'genesis_date': btc.GENESIS_DATE.isoformat(),
            'calculation_date': datetime.now().isoformat()
        },
        'milestones': {f"{pct}%": btc.time_to_percentage(pct) for pct in milestones},
        'halving_schedule': schedule,
        'comparative_analysis': comparison
    }

    output_file = '/Users/z/work/lux/ai/bitcoin_tokenomics_data.json'
    with open(output_file, 'w') as f:
        json.dump(output_data, f, indent=2, default=str)

    print(f"Data exported to: {output_file}")

    # 8. Verification
    print_section("VERIFICATION")

    # Verify 50% at first halving
    supply_at_first_halving = btc.supply_at_block(btc.HALVING_INTERVAL)
    print(f"Supply at first halving: {supply_at_first_halving:,.2f} BTC")
    print(f"Expected (50%): {btc.total_supply() / 2:,.2f} BTC")
    print(f"Match: {abs(supply_at_first_halving - btc.total_supply() / 2) < 0.01}")

    # Verify geometric series
    manual_sum = sum(btc.INITIAL_REWARD * btc.HALVING_INTERVAL / (2 ** i)
                     for i in range(btc.total_halvings))
    print(f"\nGeometric series sum: {manual_sum:,.8f} BTC")
    print(f"Formula result: {btc.total_supply():,.8f} BTC")
    print(f"Difference: {abs(manual_sum - btc.total_supply()):.10f} BTC")

    print("\n" + "="*80)
    print("Analysis complete. See bitcoin_tokenomics_analysis.md for detailed report.")
    print("="*80 + "\n")


if __name__ == "__main__":
    main()
