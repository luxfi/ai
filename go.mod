module github.com/luxfi/ai

go 1.26.4

require (
	github.com/luxfi/cc v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/google/go-sev-guest v0.14.1 // indirect
	github.com/google/logger v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
)

// Confidential-AI attestation consumes the orthogonal leaf verifier
// github.com/luxfi/cc (one verifier shared with mpc/tee, coupled to neither).
// Local replace until the leaf is tagged; ai stays free of any luxfi/mpc dep.
replace github.com/luxfi/cc => ../cc
