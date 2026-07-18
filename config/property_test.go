package config

import (
	"fmt"
	"sync"
	"testing"
)

func TestPropertyUsageValidatorConcurrentAccess(t *testing.T) {
	validator := NewPropertyUsageValidator()
	keys := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		keys = append(keys, fmt.Sprintf("key.%d", i))
	}

	var wg sync.WaitGroup
	for worker := 0; worker < 20; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for iteration := 0; iteration < 1000; iteration++ {
				key := keys[(worker+iteration)%len(keys)]
				validator.Add(key)
				_ = validator.Used(key)
				_ = validator.ValidateUnused(keys)
			}
		}(worker)
	}
	wg.Wait()
}
