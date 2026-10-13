package consumers

import (
	"testing"
)

func TestBuildConsumerRegistryDropsTheConsumersItCannotRun(t *testing.T) {
	silenceDefaultLogger(t)

	registry := BuildRegistry(RegistryDependencies{})

	if len(registry) != 0 {
		t.Errorf("registry = %d consumers, want none when no handler could be built", len(registry))
	}
}
