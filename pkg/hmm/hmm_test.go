package hmm

import (
	"math"
	"testing"
)

func newMarket() *Market {
	// 2 resources (e.g. H100-hour, RTX4090-hour) with equal starting price.
	return &Market{
		Q:        []float64{1000, 1000},
		P:        []float64{1.0, 1.0},
		Mass:     []float64{50, 50},
		Scarcity: []float64{100, 100},
	}
}

// Buying raises price; the more you buy, the higher the price (scarcity).
func TestQuoteRaisesPriceOnDemand(t *testing.T) {
	m := newMarket()
	p0 := m.P[0]
	pSmall := m.Quote(0, 10, 0.1, 1.0)
	if pSmall <= p0 {
		t.Fatalf("small buy did not raise price: %.6f -> %.6f", p0, pSmall)
	}
	m2 := newMarket()
	pLarge := m2.Quote(0, 500, 0.1, 1.0)
	if pLarge <= pSmall {
		t.Fatalf("larger buy should cost more: small=%.6f large=%.6f", pSmall, pLarge)
	}
}

// The signature symplectic property: H stays bounded (no secular drift) over
// many steps of free evolution — i.e. market liquidity is conserved.
func TestSymplecticConservesHamiltonian(t *testing.T) {
	m := newMarket()
	h0 := m.Hamiltonian()
	for i := 0; i < 100000; i++ {
		m.Step(0.001)
	}
	h1 := m.Hamiltonian()
	drift := math.Abs(h1-h0) / h0
	if drift > 1e-3 {
		t.Fatalf("symplectic integrator should conserve H; drift=%.3e (h0=%.4f h1=%.4f)", drift, h0, h1)
	}
}

// Deeper markets (more mass) have less price impact for the same trade.
func TestDepthReducesImpact(t *testing.T) {
	shallow := &Market{Q: []float64{1000}, P: []float64{1.0}, Mass: []float64{10}, Scarcity: []float64{100}}
	deep := &Market{Q: []float64{1000}, P: []float64{1.0}, Mass: []float64{1000}, Scarcity: []float64{100}}
	si := shallow.PriceImpact(0, 100, 0.1)
	di := deep.PriceImpact(0, 100, 0.1)
	if !(di < si) {
		t.Fatalf("deeper market should have lower impact: shallow=%.6f deep=%.6f", si, di)
	}
}
