// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package cc implements AI reward distribution for the Lux Network.
// This file defines the AI reward mining model per LP-5610 Section 7:
//
//   - 10% of block rewards go to AI Compute Pool
//   - 90% of block rewards go to traditional validators
//   - AI providers earn rewards for availability (random mining)
//   - Rewards scaled by CC tier, modeling level, and trust score
//
// Modeling Levels (complexity of AI work):
//   Level 1 — "Inference-Light": Embeddings, small models (<7B)
//   Level 2 — "Inference-Standard": Medium models (7B-70B), chat
//   Level 3 — "Inference-Heavy": Large models (70B+), multimodal
//   Level 4 — "Training": Fine-tuning, RLHF, distributed training
//   Level 5 — "Specialized": PQ crypto, ZK proofs, custom compute
package cc

import (
	"math/big"
	"time"
)

// AIRewardPoolShare is the percentage of block rewards allocated to AI compute
// 10% of total block rewards go to AI providers
const AIRewardPoolShare = 0.10

// ValidatorRewardShare is the percentage for traditional validators
const ValidatorRewardShare = 0.90

// ModelingLevel represents the complexity tier of AI workloads
type ModelingLevel uint8

const (
	// ModelingLevelInferenceLight is for embeddings and small models (<7B params)
	// Examples: text-embedding-3, Phi-3-mini, Qwen3-0.6B
	ModelingLevelInferenceLight ModelingLevel = 1

	// ModelingLevelInferenceStandard is for medium models (7B-70B params)
	// Examples: Llama-3-8B, Qwen3-14B, Mistral-7B
	ModelingLevelInferenceStandard ModelingLevel = 2

	// ModelingLevelInferenceHeavy is for large models (70B+ params)
	// Examples: Llama-3-70B, Qwen3-72B, multimodal models
	ModelingLevelInferenceHeavy ModelingLevel = 3

	// ModelingLevelTraining is for fine-tuning and training workloads
	// Examples: LoRA, QLoRA, full fine-tuning, RLHF
	ModelingLevelTraining ModelingLevel = 4

	// ModelingLevelSpecialized is for specialized compute
	// Examples: PQ crypto operations, ZK proof generation, custom kernels
	ModelingLevelSpecialized ModelingLevel = 5
)

// String returns the human-readable name for the modeling level
func (l ModelingLevel) String() string {
	switch l {
	case ModelingLevelInferenceLight:
		return "Inference-Light"
	case ModelingLevelInferenceStandard:
		return "Inference-Standard"
	case ModelingLevelInferenceHeavy:
		return "Inference-Heavy"
	case ModelingLevelTraining:
		return "Training"
	case ModelingLevelSpecialized:
		return "Specialized"
	default:
		return "Unknown"
	}
}

// BaseRewardMultiplier returns the base reward multiplier for the modeling level
// Higher complexity = higher rewards
func (l ModelingLevel) BaseRewardMultiplier() float64 {
	switch l {
	case ModelingLevelInferenceLight:
		return 0.5
	case ModelingLevelInferenceStandard:
		return 1.0
	case ModelingLevelInferenceHeavy:
		return 1.5
	case ModelingLevelTraining:
		return 2.0
	case ModelingLevelSpecialized:
		return 2.5
	default:
		return 0.0
	}
}

// MinVRAMGB returns minimum VRAM required for this modeling level
func (l ModelingLevel) MinVRAMGB() uint64 {
	switch l {
	case ModelingLevelInferenceLight:
		return 8 // 8GB for small models
	case ModelingLevelInferenceStandard:
		return 24 // 24GB for 7B-13B models
	case ModelingLevelInferenceHeavy:
		return 80 // 80GB for 70B+ models
	case ModelingLevelTraining:
		return 48 // 48GB minimum for training
	case ModelingLevelSpecialized:
		return 16 // Varies, 16GB baseline
	default:
		return 0
	}
}

// AIProvider represents an AI compute provider in the reward pool
type AIProvider struct {
	// ProviderID is the unique identifier
	ProviderID string `json:"provider_id"`

	// Attestation is the current CC tier attestation
	Attestation *TierAttestation `json:"attestation"`

	// MaxModelingLevel is the highest modeling level supported
	MaxModelingLevel ModelingLevel `json:"max_modeling_level"`

	// CurrentModelingLevel is the current active workload level
	CurrentModelingLevel ModelingLevel `json:"current_modeling_level"`

	// StakeLUX is the staked amount in LUX tokens
	StakeLUX uint64 `json:"stake_lux"`

	// LastHeartbeat is when the provider last checked in
	LastHeartbeat time.Time `json:"last_heartbeat"`

	// ConsecutiveEpochs is consecutive epochs online
	ConsecutiveEpochs uint64 `json:"consecutive_epochs"`

	// TasksThisEpoch is tasks completed in current epoch
	TasksThisEpoch uint64 `json:"tasks_this_epoch"`

	// TotalTasksCompleted is lifetime tasks completed
	TotalTasksCompleted uint64 `json:"total_tasks_completed"`

	// ReputationScore is 0.0-1.0 historical reputation
	ReputationScore float64 `json:"reputation_score"`
}

// IsOnline checks if the provider is currently online
func (p *AIProvider) IsOnline(maxHeartbeatAge time.Duration) bool {
	return time.Since(p.LastHeartbeat) < maxHeartbeatAge
}

// EffectiveTier returns the CC tier from attestation, or Tier4 if none
func (p *AIProvider) EffectiveTier() CCTier {
	if p.Attestation != nil && p.Attestation.IsValid() {
		return p.Attestation.Tier
	}
	return Tier4Standard
}

// RewardWeight calculates the provider's weight in the reward pool
// Weight = TierMultiplier * ModelingMultiplier * StakeWeight * UptimeBonus * ReputationBonus
func (p *AIProvider) RewardWeight() float64 {
	tier := p.EffectiveTier()

	// Base tier multiplier (1.5x for Tier1, down to 0.5x for Tier4)
	tierMult := tier.RewardMultiplier()

	// Modeling level multiplier
	modelMult := p.MaxModelingLevel.BaseRewardMultiplier()

	// Stake weight (logarithmic to prevent plutocracy)
	// sqrt(stake / 1000) capped at 10x
	stakeWeight := 1.0
	if p.StakeLUX > 1000 {
		stakeWeight = min(10.0, sqrt(float64(p.StakeLUX)/1000.0))
	}

	// Uptime bonus (up to 1.5x for long-term providers)
	uptimeBonus := 1.0 + min(0.5, float64(p.ConsecutiveEpochs)/1000.0)

	// Reputation bonus (0.8x to 1.2x based on history)
	repBonus := 0.8 + (p.ReputationScore * 0.4)

	return tierMult * modelMult * stakeWeight * uptimeBonus * repBonus
}

// sqrt is a simple integer square root approximation
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// min returns the smaller of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// AIRewardPool manages the AI compute reward distribution
type AIRewardPool struct {
	// Providers is the list of registered AI providers
	Providers map[string]*AIProvider `json:"providers"`

	// EpochNumber is the current reward epoch
	EpochNumber uint64 `json:"epoch_number"`

	// EpochDuration is the length of each reward epoch
	EpochDuration time.Duration `json:"epoch_duration"`

	// TotalPoolLUX is the total LUX in the AI reward pool for this epoch
	TotalPoolLUX *big.Int `json:"total_pool_lux"`

	// ParticipationShare is the % of AI pool for random availability rewards
	// Default: 30% of AI pool (3% of total block rewards)
	ParticipationShare float64 `json:"participation_share"`

	// TaskShare is the % of AI pool for task completion rewards
	// Default: 70% of AI pool (7% of total block rewards)
	TaskShare float64 `json:"task_share"`
}

// NewAIRewardPool creates a new AI reward pool
func NewAIRewardPool(epochDuration time.Duration) *AIRewardPool {
	return &AIRewardPool{
		Providers:          make(map[string]*AIProvider),
		EpochDuration:      epochDuration,
		TotalPoolLUX:       big.NewInt(0),
		ParticipationShare: 0.30, // 30% for availability
		TaskShare:          0.70, // 70% for tasks
	}
}

// RegisterProvider adds a provider to the pool
func (pool *AIRewardPool) RegisterProvider(provider *AIProvider) error {
	if provider.ProviderID == "" {
		return ErrInvalidAttestation
	}
	if provider.StakeLUX < Tier4Standard.MinStakeLUX() {
		return ErrInsufficientStake
	}
	pool.Providers[provider.ProviderID] = provider
	return nil
}

// CalculateBlockRewardSplit splits block reward between validators and AI pool
func CalculateBlockRewardSplit(totalBlockReward *big.Int) (validatorReward, aiPoolReward *big.Int) {
	// 90% to validators
	validatorReward = new(big.Int).Mul(totalBlockReward, big.NewInt(90))
	validatorReward.Div(validatorReward, big.NewInt(100))

	// 10% to AI pool
	aiPoolReward = new(big.Int).Sub(totalBlockReward, validatorReward)

	return validatorReward, aiPoolReward
}

// ParticipationRewardResult contains the participation reward calculation
type ParticipationRewardResult struct {
	// ProviderID is the provider receiving the reward
	ProviderID string `json:"provider_id"`

	// RewardLUX is the reward amount in LUX (wei)
	RewardLUX *big.Int `json:"reward_lux"`

	// Weight is the provider's calculated weight
	Weight float64 `json:"weight"`

	// WeightShare is the provider's share of total weight
	WeightShare float64 `json:"weight_share"`

	// Tier is the provider's CC tier
	Tier CCTier `json:"tier"`

	// ModelingLevel is the provider's max modeling level
	ModelingLevel ModelingLevel `json:"modeling_level"`
}

// CalculateParticipationRewards distributes the participation pool
// This is the "random mining" reward - providers earn just for being online and attested
func (pool *AIRewardPool) CalculateParticipationRewards(
	maxHeartbeatAge time.Duration,
) []*ParticipationRewardResult {
	// Get participation pool amount
	participationPool := new(big.Int).Set(pool.TotalPoolLUX)
	participationPool.Mul(participationPool, big.NewInt(int64(pool.ParticipationShare*100)))
	participationPool.Div(participationPool, big.NewInt(100))

	// Calculate total weight of online providers
	var totalWeight float64
	onlineProviders := make([]*AIProvider, 0)

	for _, provider := range pool.Providers {
		if !provider.IsOnline(maxHeartbeatAge) {
			continue
		}
		if provider.Attestation == nil || !provider.Attestation.IsValid() {
			continue
		}
		weight := provider.RewardWeight()
		totalWeight += weight
		onlineProviders = append(onlineProviders, provider)
	}

	if totalWeight == 0 || len(onlineProviders) == 0 {
		return nil
	}

	// Distribute rewards proportionally to weight
	results := make([]*ParticipationRewardResult, 0, len(onlineProviders))

	for _, provider := range onlineProviders {
		weight := provider.RewardWeight()
		share := weight / totalWeight

		reward := new(big.Int).Set(participationPool)
		reward.Mul(reward, big.NewInt(int64(share*1e9)))
		reward.Div(reward, big.NewInt(1e9))

		results = append(results, &ParticipationRewardResult{
			ProviderID:    provider.ProviderID,
			RewardLUX:     reward,
			Weight:        weight,
			WeightShare:   share,
			Tier:          provider.EffectiveTier(),
			ModelingLevel: provider.MaxModelingLevel,
		})
	}

	return results
}

// TaskRewardResult contains the task completion reward calculation
type TaskRewardResult struct {
	// ProviderID is the provider receiving the reward
	ProviderID string `json:"provider_id"`

	// TaskID is the completed task identifier
	TaskID string `json:"task_id"`

	// RewardLUX is the reward amount in LUX (wei)
	RewardLUX *big.Int `json:"reward_lux"`

	// ModelingLevel is the task's modeling level
	ModelingLevel ModelingLevel `json:"modeling_level"`

	// ComputeUnits is the compute units consumed
	ComputeUnits uint64 `json:"compute_units"`
}

// CalculateTaskReward calculates reward for a completed task
func (pool *AIRewardPool) CalculateTaskReward(
	provider *AIProvider,
	taskID string,
	modelingLevel ModelingLevel,
	computeUnits uint64,
) *TaskRewardResult {
	// Base rate per compute unit (in wei)
	// 1 compute unit = 1 GPU-second at Tier 2 / Level 2
	baseRateWei := big.NewInt(1e12) // 0.000001 LUX per compute unit

	// Calculate reward
	reward := new(big.Int).Mul(baseRateWei, big.NewInt(int64(computeUnits)))

	// Apply tier multiplier
	tierMult := provider.EffectiveTier().RewardMultiplier()
	reward.Mul(reward, big.NewInt(int64(tierMult*100)))
	reward.Div(reward, big.NewInt(100))

	// Apply modeling level multiplier
	levelMult := modelingLevel.BaseRewardMultiplier()
	reward.Mul(reward, big.NewInt(int64(levelMult*100)))
	reward.Div(reward, big.NewInt(100))

	return &TaskRewardResult{
		ProviderID:    provider.ProviderID,
		TaskID:        taskID,
		RewardLUX:     reward,
		ModelingLevel: modelingLevel,
		ComputeUnits:  computeUnits,
	}
}

// EpochRewardSummary contains the full epoch reward distribution
type EpochRewardSummary struct {
	// EpochNumber is the epoch being summarized
	EpochNumber uint64 `json:"epoch_number"`

	// TotalBlockRewardsLUX is total block rewards in the epoch
	TotalBlockRewardsLUX *big.Int `json:"total_block_rewards_lux"`

	// ValidatorRewardsLUX is 90% going to validators
	ValidatorRewardsLUX *big.Int `json:"validator_rewards_lux"`

	// AIPoolRewardsLUX is 10% going to AI providers
	AIPoolRewardsLUX *big.Int `json:"ai_pool_rewards_lux"`

	// ParticipationRewardsLUX is 30% of AI pool (3% total)
	ParticipationRewardsLUX *big.Int `json:"participation_rewards_lux"`

	// TaskRewardsLUX is 70% of AI pool (7% total)
	TaskRewardsLUX *big.Int `json:"task_rewards_lux"`

	// OnlineProviders is count of providers that were online
	OnlineProviders uint64 `json:"online_providers"`

	// TotalProviders is count of all registered providers
	TotalProviders uint64 `json:"total_providers"`

	// ProviderRewards is the per-provider reward breakdown
	ProviderRewards []*ParticipationRewardResult `json:"provider_rewards"`

	// TierDistribution shows providers by tier
	TierDistribution map[CCTier]uint64 `json:"tier_distribution"`
}

// CalculateEpochRewards calculates full epoch reward distribution
func (pool *AIRewardPool) CalculateEpochRewards(
	totalBlockRewards *big.Int,
	maxHeartbeatAge time.Duration,
) *EpochRewardSummary {
	validatorRewards, aiPoolRewards := CalculateBlockRewardSplit(totalBlockRewards)

	// Update pool total
	pool.TotalPoolLUX = aiPoolRewards

	// Calculate participation rewards
	participationRewards := pool.CalculateParticipationRewards(maxHeartbeatAge)

	// Calculate pool splits
	participationPool := new(big.Int).Set(aiPoolRewards)
	participationPool.Mul(participationPool, big.NewInt(int64(pool.ParticipationShare*100)))
	participationPool.Div(participationPool, big.NewInt(100))

	taskPool := new(big.Int).Sub(aiPoolRewards, participationPool)

	// Count tiers
	tierDist := make(map[CCTier]uint64)
	var onlineCount uint64
	for _, provider := range pool.Providers {
		if provider.IsOnline(maxHeartbeatAge) {
			onlineCount++
			tier := provider.EffectiveTier()
			tierDist[tier]++
		}
	}

	return &EpochRewardSummary{
		EpochNumber:             pool.EpochNumber,
		TotalBlockRewardsLUX:    totalBlockRewards,
		ValidatorRewardsLUX:     validatorRewards,
		AIPoolRewardsLUX:        aiPoolRewards,
		ParticipationRewardsLUX: participationPool,
		TaskRewardsLUX:          taskPool,
		OnlineProviders:         onlineCount,
		TotalProviders:          uint64(len(pool.Providers)),
		ProviderRewards:         participationRewards,
		TierDistribution:        tierDist,
	}
}

// RandomMiningEligibility checks if a provider is eligible for random mining rewards
func RandomMiningEligibility(provider *AIProvider, maxHeartbeatAge time.Duration) (bool, string) {
	if provider == nil {
		return false, "provider is nil"
	}

	if !provider.IsOnline(maxHeartbeatAge) {
		return false, "provider offline"
	}

	if provider.Attestation == nil {
		return false, "no attestation"
	}

	if !provider.Attestation.IsValid() {
		return false, "attestation expired"
	}

	minStake := provider.EffectiveTier().MinStakeLUX()
	if provider.StakeLUX < minStake {
		return false, "insufficient stake"
	}

	return true, "eligible"
}
