#!/usr/bin/env python3
"""
Bitcoin Tokenomics Visualization
Creates charts showing supply distribution and inflation rates over time
"""

import json
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime
import numpy as np


def load_data():
    """Load the calculated tokenomics data."""
    with open('/Users/z/work/lux/ai/bitcoin_tokenomics_data.json', 'r') as f:
        return json.load(f)


def plot_supply_curve(data):
    """
    Plot Bitcoin supply accumulation over time.
    """
    schedule = data['halving_schedule']

    # Extract data
    years = [event['years_from_genesis'] for event in schedule]
    supply = [event['cumulative_supply_btc'] for event in schedule]

    # Create figure
    fig, ax = plt.subplots(figsize=(14, 8))

    # Plot supply curve
    ax.plot(years, supply, linewidth=2.5, color='#F7931A', label='Cumulative Supply')

    # Add milestone markers
    milestones = [
        (4, 10500000, '50%'),
        (8, 15750000, '75%'),
        (13.6, 18900000, '90%'),
        (27, 20790000, '99%'),
        (40, 20979000, '99.9%')
    ]

    for year, supply_val, label in milestones:
        ax.plot(year, supply_val, 'ro', markersize=8)
        ax.annotate(label, xy=(year, supply_val), xytext=(year+2, supply_val),
                   fontsize=10, color='red', fontweight='bold')

    # Add 21M cap line
    ax.axhline(y=21000000, color='black', linestyle='--', linewidth=2,
               label='21M Hard Cap', alpha=0.7)

    # Formatting
    ax.set_xlabel('Years from Genesis (2009)', fontsize=12, fontweight='bold')
    ax.set_ylabel('Cumulative Supply (BTC)', fontsize=12, fontweight='bold')
    ax.set_title('Bitcoin Supply Distribution Over Time', fontsize=16, fontweight='bold')
    ax.legend(fontsize=11, loc='lower right')
    ax.grid(True, alpha=0.3)
    ax.set_xlim(0, 60)
    ax.set_ylim(0, 22000000)

    # Format y-axis
    ax.yaxis.set_major_formatter(plt.FuncFormatter(lambda x, p: f'{int(x/1e6)}M'))

    plt.tight_layout()
    plt.savefig('/Users/z/work/lux/ai/bitcoin_supply_curve.png', dpi=300, bbox_inches='tight')
    print("Supply curve saved to: /Users/z/work/lux/ai/bitcoin_supply_curve.png")


def plot_inflation_rates(data):
    """
    Plot inflation rate decay over time.
    """
    schedule = data['halving_schedule']

    # Extract data (skip epoch 0 which has infinite inflation)
    valid_events = [e for e in schedule[1:30] if e['inflation_rate_percent'] != 'N/A']
    years = [event['years_from_genesis'] for event in valid_events]
    inflation = [event['inflation_rate_percent'] for event in valid_events]

    # Create figure with two subplots
    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(14, 10))

    # Plot 1: Linear scale (first 40 years)
    mask = np.array(years) <= 40
    ax1.plot(np.array(years)[mask], np.array(inflation)[mask],
             linewidth=2.5, color='#F7931A', marker='o', markersize=6)

    # Add reference lines
    ax1.axhline(y=2.0, color='green', linestyle='--', linewidth=1.5,
                label='Fed Target (2%)', alpha=0.7)
    ax1.axhline(y=1.5, color='brown', linestyle='--', linewidth=1.5,
                label='Gold (~1.5%)', alpha=0.7)

    ax1.set_xlabel('Years from Genesis (2009)', fontsize=12, fontweight='bold')
    ax1.set_ylabel('Annual Inflation Rate (%)', fontsize=12, fontweight='bold')
    ax1.set_title('Bitcoin Inflation Rate Decay (Linear Scale)', fontsize=14, fontweight='bold')
    ax1.legend(fontsize=10)
    ax1.grid(True, alpha=0.3)
    ax1.set_xlim(0, 40)

    # Plot 2: Log scale (all epochs)
    ax2.semilogy(years, inflation, linewidth=2.5, color='#F7931A',
                 marker='o', markersize=5)

    ax2.axhline(y=2.0, color='green', linestyle='--', linewidth=1.5,
                label='Fed Target (2%)', alpha=0.7)
    ax2.axhline(y=1.5, color='brown', linestyle='--', linewidth=1.5,
                label='Gold (~1.5%)', alpha=0.7)

    ax2.set_xlabel('Years from Genesis (2009)', fontsize=12, fontweight='bold')
    ax2.set_ylabel('Annual Inflation Rate (%, log scale)', fontsize=12, fontweight='bold')
    ax2.set_title('Bitcoin Inflation Rate Decay (Logarithmic Scale)', fontsize=14, fontweight='bold')
    ax2.legend(fontsize=10)
    ax2.grid(True, alpha=0.3, which='both')
    ax2.set_xlim(0, 120)

    plt.tight_layout()
    plt.savefig('/Users/z/work/lux/ai/bitcoin_inflation_rates.png', dpi=300, bbox_inches='tight')
    print("Inflation rates saved to: /Users/z/work/lux/ai/bitcoin_inflation_rates.png")


def plot_issuance_schedule(data):
    """
    Plot block reward and annual issuance over time.
    """
    schedule = data['halving_schedule']

    years = [event['years_from_genesis'] for event in schedule[:20]]
    rewards = [event['reward_btc'] for event in schedule[:20]]
    annual_issuance = [event['annual_issuance_btc'] for event in schedule[:20]]

    # Create figure with two y-axes
    fig, ax1 = plt.subplots(figsize=(14, 8))

    color1 = '#F7931A'
    ax1.set_xlabel('Years from Genesis (2009)', fontsize=12, fontweight='bold')
    ax1.set_ylabel('Block Reward (BTC)', fontsize=12, fontweight='bold', color=color1)
    ax1.semilogy(years, rewards, color=color1, linewidth=2.5, marker='o',
                 markersize=7, label='Block Reward')
    ax1.tick_params(axis='y', labelcolor=color1)
    ax1.grid(True, alpha=0.3, which='both')

    # Second y-axis
    ax2 = ax1.twinx()
    color2 = 'darkblue'
    ax2.set_ylabel('Annual Issuance (BTC)', fontsize=12, fontweight='bold', color=color2)
    ax2.semilogy(years, annual_issuance, color=color2, linewidth=2.5,
                 marker='s', markersize=6, linestyle='--', label='Annual Issuance')
    ax2.tick_params(axis='y', labelcolor=color2)

    # Title
    ax1.set_title('Bitcoin Block Reward and Annual Issuance Schedule',
                  fontsize=16, fontweight='bold')

    # Legends
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc='upper right', fontsize=11)

    plt.tight_layout()
    plt.savefig('/Users/z/work/lux/ai/bitcoin_issuance_schedule.png', dpi=300, bbox_inches='tight')
    print("Issuance schedule saved to: /Users/z/work/lux/ai/bitcoin_issuance_schedule.png")


def plot_security_budget(data):
    """
    Plot security budget projection.
    """
    schedule = data['halving_schedule']

    years = [event['years_from_genesis'] for event in schedule[:25]]
    btc_price = 100000  # Assume $100k BTC

    # Calculate security budget at different BTC prices
    rewards = [event['annual_issuance_btc'] for event in schedule[:25]]

    budget_50k = [r * 50000 / 1e9 for r in rewards]  # In billions USD
    budget_100k = [r * 100000 / 1e9 for r in rewards]
    budget_200k = [r * 200000 / 1e9 for r in rewards]

    # Create figure
    fig, ax = plt.subplots(figsize=(14, 8))

    ax.semilogy(years, budget_50k, linewidth=2.5, label='BTC = $50k', marker='o')
    ax.semilogy(years, budget_100k, linewidth=2.5, label='BTC = $100k', marker='s')
    ax.semilogy(years, budget_200k, linewidth=2.5, label='BTC = $200k', marker='^')

    # Add critical threshold line (estimated)
    ax.axhline(y=10, color='red', linestyle='--', linewidth=2,
               label='Est. Min Security Budget ($10B/yr)', alpha=0.7)

    ax.set_xlabel('Years from Genesis (2009)', fontsize=12, fontweight='bold')
    ax.set_ylabel('Annual Security Budget (Billions USD, log scale)',
                  fontsize=12, fontweight='bold')
    ax.set_title('Bitcoin Security Budget Projection (Block Rewards Only)',
                 fontsize=16, fontweight='bold')
    ax.legend(fontsize=11, loc='upper right')
    ax.grid(True, alpha=0.3, which='both')
    ax.set_xlim(0, 100)

    plt.tight_layout()
    plt.savefig('/Users/z/work/lux/ai/bitcoin_security_budget.png', dpi=300, bbox_inches='tight')
    print("Security budget saved to: /Users/z/work/lux/ai/bitcoin_security_budget.png")


def plot_supply_distribution(data):
    """
    Plot distribution of supply across epochs.
    """
    schedule = data['halving_schedule']

    epochs = [event['halving'] for event in schedule[:10]]
    supply_per_epoch = []

    for i in range(10):
        if i == 0:
            supply = schedule[i+1]['cumulative_supply_btc']
        else:
            supply = (schedule[i+1]['cumulative_supply_btc'] -
                     schedule[i]['cumulative_supply_btc'])
        supply_per_epoch.append(supply)

    # Create pie chart
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 8))

    # Pie chart
    colors = plt.cm.YlOrRd(np.linspace(0.3, 0.9, 10))
    wedges, texts, autotexts = ax1.pie(supply_per_epoch, labels=[f'Epoch {i}' for i in epochs],
                                        autopct='%1.1f%%', startangle=90, colors=colors,
                                        textprops={'fontsize': 10})

    ax1.set_title('Bitcoin Supply Distribution by Epoch\n(First 10 Halvings)',
                  fontsize=14, fontweight='bold')

    # Bar chart
    ax2.bar(epochs, supply_per_epoch, color=colors, edgecolor='black', linewidth=1.5)
    ax2.set_xlabel('Halving Epoch', fontsize=12, fontweight='bold')
    ax2.set_ylabel('BTC Issued in Epoch (Millions)', fontsize=12, fontweight='bold')
    ax2.set_title('BTC Issuance by Epoch', fontsize=14, fontweight='bold')
    ax2.yaxis.set_major_formatter(plt.FuncFormatter(lambda x, p: f'{x/1e6:.1f}M'))
    ax2.grid(True, alpha=0.3, axis='y')

    plt.tight_layout()
    plt.savefig('/Users/z/work/lux/ai/bitcoin_supply_distribution.png', dpi=300, bbox_inches='tight')
    print("Supply distribution saved to: /Users/z/work/lux/ai/bitcoin_supply_distribution.png")


def main():
    """Generate all visualizations."""
    print("Loading Bitcoin tokenomics data...")
    data = load_data()

    print("\nGenerating visualizations...\n")

    plot_supply_curve(data)
    plot_inflation_rates(data)
    plot_issuance_schedule(data)
    plot_security_budget(data)
    plot_supply_distribution(data)

    print("\nAll visualizations complete!")
    print("\nGenerated files:")
    print("  - bitcoin_supply_curve.png")
    print("  - bitcoin_inflation_rates.png")
    print("  - bitcoin_issuance_schedule.png")
    print("  - bitcoin_security_budget.png")
    print("  - bitcoin_supply_distribution.png")


if __name__ == "__main__":
    main()
