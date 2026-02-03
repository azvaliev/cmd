package main

import (
	"context"
	"fmt"
	"os"

	"github.com/azvaliev/cmd/internal/pkg/ai"
)

func main() {
	server, err := ai.CreateLLamaServer(ai.IBM_GRANITE_MODEL_CONFIG)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Dispose()

	a := ai.NewCommandAgent(server, context.Background())

	out, err := a.Generate("find all files >100 MB")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(out)

}
