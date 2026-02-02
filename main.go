package main

import (
	"fmt"
)

func main() {
	server, err := CreateLLamaServer(QWEN_25_CODER_MODEL_CONFIG)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Dispose()

	fmt.Println("Server started")
}
