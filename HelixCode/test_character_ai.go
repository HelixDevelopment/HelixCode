package main

import (
	"dev.helix.code/internal/memory/providers"
	"fmt"
)

func main() {
	config := map[string]interface{}{
		"api_key": "test_key",
	}

	provider, err := providers.NewCharacterAIProvider(config)
	if err != nil {
		fmt.Printf("Failed to create provider: %v\n", err)
		return
	}

	fmt.Printf("Provider created successfully: %s\n", provider.GetName())
	fmt.Printf("Provider type: %s\n", provider.GetType())
}
