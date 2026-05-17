package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIngestRecord(t *testing.T) {
	rec := IngestRecord{
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Now(),
		Quality:    "good",
	}
	assert.Equal(t, "heart_rate", rec.MetricType)
}