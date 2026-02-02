package main

import (
	"fmt"

	"github.com/azvaliev/cmd/internal/pkg/ai"
)

func main() {
	server, err := ai.CreateLLamaServer(ai.QWEN_25_CODER_MODEL_CONFIG)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Dispose()

	fmt.Println("Server started")
}
