# ADR 0001: Unified Wearable Device Integration and Biometric Data Aggregation

## Context

The project requires robust integration with wearable devices while avoiding direct device-specific APIs. The architecture should support devices from multiple ecosystems and normalize biometric inputs for downstream ML and planning services.

## Decision

Use an adapter-based aggregation layer that:

- abstracts vendor/platform-specific sources behind `BiometricSource`
- supports both device-level adapters (`apple`, `samsung`, `huawei`, `amazfit`) and unified platform adapters (`rook`, `terra`, `health_connect`, `healthkit`)
- aggregates data from multiple sources with graceful degradation
- normalizes duplicate metrics by quality/confidence and attaches source metadata

## Consequences

- allows plug-and-play integration of new wearable platforms
- supports the recommended strategy of aggregation through unified platforms
- supports fallback to emulator/mock sources for testing
- makes biometric ingestion resilient to partial source failures

## Implementation

- added `internal/biometric/adapters/aggregator.go`
- added platform adapter stubs in `internal/biometric/adapters/vendor_adapters.go`
- added unit tests in `internal/biometric/adapters/aggregator_test.go`

