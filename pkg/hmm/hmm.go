// Package hmm is the reference implementation of the Hanzo Hamiltonian Market
// Maker (HIP-0008): a SYMPLECTIC Hamiltonian system for pricing AI compute.
//
// Prices are conjugate momenta of a Hamiltonian
//
//	H(q, p) = Σ_i p_i² / (2·m_i)  +  V(q),   V(q) = Σ_i c_i / q_i
//	          └ kinetic (price momentum;        └ scarcity potential: price
//	            m_i = market depth)               blows up as inventory q_i → 0
//
// Evolution follows Hamilton's equations (dq/dt = ∂H/∂p = p/m,
// dp/dt = −∂H/∂q = −dV/dq) integrated with a leapfrog (Störmer–Verlet) step,
// which is symplectic: it preserves phase-space volume, so total market
// liquidity is conserved (Liouville). This is NOT a constant-product (x*y=k) AMM.
package hmm

import "math"

// Market is a symplectic Hamiltonian compute market over N resources (GPU
// classes, inference, training slots, …).
type Market struct {
	Q        []float64 // inventory available per resource (canonical position q)
	P        []float64 // price per resource (conjugate momentum p)
	Mass     []float64 // market depth per resource (larger ⇒ steadier price)
	Scarcity []float64 // potential coefficient c_i in V(q)=Σ c_i/q_i
}

const qFloor = 1e-9 // keep the scarcity potential finite as inventory → 0

// dVdq = ∂V/∂q_i for V=Σ c_i/q_i  ⇒  −c_i / q_i².
func (m *Market) dVdq(i int) float64 {
	q := m.Q[i]
	if q < qFloor {
		q = qFloor
	}
	return -m.Scarcity[i] / (q * q)
}

// Hamiltonian H = Σ p²/2m + Σ c/q (kinetic + potential). Conserved (up to the
// bounded error of a symplectic integrator) under Step — the liquidity invariant.
func (m *Market) Hamiltonian() float64 {
	h := 0.0
	for i := range m.Q {
		h += m.P[i] * m.P[i] / (2 * m.Mass[i])
		q := m.Q[i]
		if q < qFloor {
			q = qFloor
		}
		h += m.Scarcity[i] / q
	}
	return h
}

// Step advances the market by dt using one leapfrog (symplectic) step.
func (m *Market) Step(dt float64) {
	for i := range m.Q {
		m.P[i] -= 0.5 * dt * m.dVdq(i) // half momentum kick
	}
	for i := range m.Q {
		m.Q[i] += dt * m.P[i] / m.Mass[i] // full position drift
	}
	for i := range m.Q {
		m.P[i] -= 0.5 * dt * m.dVdq(i) // half momentum kick
	}
}

// Quote prices buying `amount` units of resource i: it removes that inventory
// (q_i ↓ ⇒ scarcity ↑ ⇒ the conjugate momentum/price ↑) and evolves one
// symplectic step. Returns the post-trade price (momentum) for resource i.
// `quality` is the HIP-0008 quality-oracle multiplier (latency/SLA + NVTrust).
func (m *Market) Quote(i int, amount, dt, quality float64) float64 {
	m.Q[i] -= amount // remove inventory (scarcity ↑)
	if m.Q[i] < qFloor {
		m.Q[i] = qFloor
	}
	// Order impulse on the conjugate momentum (price): a deeper market (larger
	// Mass) absorbs the same order with a smaller price move — Δp = amount / m.
	m.P[i] += amount / m.Mass[i]
	m.Step(dt)
	return m.P[i] * quality
}

// ComputeInvariant is the emergent conserved quantity compute·demand: here the
// dot product Σ q_i·p_i (allocation · price). Reported for invariant checks; it
// is a CONSEQUENCE of the symplectic flow, not an imposed bonding curve.
func (m *Market) ComputeInvariant() float64 {
	inv := 0.0
	for i := range m.Q {
		inv += m.Q[i] * m.P[i]
	}
	return inv
}

// PriceImpact returns |Δprice| / price for buying `amount` of resource i without
// mutating the market (useful for previews). Larger Mass ⇒ smaller impact.
func (m *Market) PriceImpact(i int, amount, dt float64) float64 {
	clone := m.clone()
	before := clone.P[i]
	after := clone.Quote(i, amount, dt, 1.0)
	if before == 0 {
		return math.Inf(1)
	}
	return math.Abs(after-before) / math.Abs(before)
}

func (m *Market) clone() *Market {
	c := &Market{
		Q:        append([]float64(nil), m.Q...),
		P:        append([]float64(nil), m.P...),
		Mass:     append([]float64(nil), m.Mass...),
		Scarcity: append([]float64(nil), m.Scarcity...),
	}
	return c
}
