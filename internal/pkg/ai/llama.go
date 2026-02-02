package ai

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/azvaliev/cmd/internal/pkg/env"
)

type ModelConfig struct {
	ModelPath string
	// -1 for unlimited
	ReasoningBudget int
	Temperature     float64
	MinP            float64
	TopP            float64
	TopK            int
	FlashAttn       bool
	RepeatPenalty   float64
}

// TODO: I should be able to get away with Q6 on these models, and Q4 on LFM 2.5

// Qwen 3.0 1.7B
//
// https://huggingface.co/Qwen/Qwen3-1.7B
var QWEN_3_MODEL_CONFIG ModelConfig = ModelConfig{
	ModelPath:       "/Users/azatvaliev/.lmstudio/models/unsloth/Qwen3-1.7B-GGUF/Qwen3-1.7B-Q8_0.gguf",
	ReasoningBudget: 0,
	Temperature:     0.7,
	MinP:            0.01,
	TopP:            0.9,
	TopK:            20,
	FlashAttn:       true,
}

// LFM 2.5 1.2B Instruct
//
// https://huggingface.co/LiquidAI/LFM2.5-1.2B-Instruct
var LFM_25_INSTRUCT_MODEL_CONFIG ModelConfig = ModelConfig{
	ModelPath:       "/Users/azatvaliev/.lmstudio/models/LiquidAI/LFM2.5-1.2B-Instruct-GGUF/LFM2.5-1.2B-Instruct-Q8_0.gguf",
	ReasoningBudget: 0,
	Temperature:     0.1,
	MinP:            0.15,
	FlashAttn:       false,
	RepeatPenalty:   1.05,
}

// Qwen 2.5 Coder 1.2B
//
// https://huggingface.co/Qwen/Qwen2.5-Coder-1.5B-Instruct
var QWEN_25_CODER_MODEL_CONFIG ModelConfig = ModelConfig{
	ModelPath:       "/Users/azatvaliev/.lmstudio/models/Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF/qwen2.5-coder-1.5b-instruct-q8_0.gguf",
	ReasoningBudget: 0,
	Temperature:     0.7,
	MinP:            0.01,
	TopP:            0.9,
	TopK:            20,
	FlashAttn:       true,
}

// IBM Granite 4.0 H 1b
//
// https://huggingface.co/ibm-granite/granite-4.0-h-1b
var IBM_GRANITE_MODEL_CONFIG ModelConfig = ModelConfig{
	ModelPath:       "/Users/azatvaliev/.lmstudio/models/ibm-granite/granite-4.0-h-1b-GGUF/granite-4.0-h-1b-Q8_0.gguf",
	ReasoningBudget: 0,
	Temperature:     0.0,
	TopP:            1.0,
	TopK:            0,
	FlashAttn:       false,
}

type LlamaServer struct {
	ModelConfig
	cmd  *exec.Cmd
	port int
}

func (llamaServer *LlamaServer) Dispose() {
	if llamaServer.cmd.Process == nil {
		return
	}

	llamaServer.cmd.Process.Signal(syscall.SIGTERM)
}

func (llamaServer *LlamaServer) HealthCheck() error {
	res, err := http.Get(
		fmt.Sprintf("http://localhost:%d/health", llamaServer.port),
	)

	if res != nil && res.StatusCode == http.StatusOK {
		return nil
	} else if res != nil {
		return fmt.Errorf("healthcheck returned bad status code: %d", res.StatusCode)
	} else {
		return errors.Join(
			fmt.Errorf("healthcheck failed"),
			err,
		)
	}
}

func CreateLLamaServer(modelConfig ModelConfig) (*LlamaServer, error) {
	port, err := GetFreePort()
	if err != nil {
		return nil, errors.Join(
			err,
			errors.New("Failed to get free port"),
		)
	}

	args := []string{
		"--no-webui",
		"--model",
		modelConfig.ModelPath,
		"--port",
		fmt.Sprintf("%d", port),
		// 4096 token context
		"--ctx-size",
		"4096",
		// full offload to GPU
		"--n-gpu-layers",
		"999",
		"--batch-size",
		"2048",
		"--ubatch-size",
		"512",
	}

	args = append(args,
		"--reasoning-budget",
		fmt.Sprintf("%d", modelConfig.ReasoningBudget),
	)

	args = append(args,
		"--temp",
		fmt.Sprintf("%f", modelConfig.Temperature),
	)

	if modelConfig.FlashAttn {
		args = append(args,
			"--flash-attn",
			"on",
		)
	} else {
		args = append(args,
			"--flash-attn",
			"off",
		)
	}

	if modelConfig.MinP != 0 {
		args = append(args,
			"--min-p",
			fmt.Sprintf("%f", modelConfig.MinP),
		)
	}

	if modelConfig.TopP != 0 {
		args = append(args,
			"--top-p",
			fmt.Sprintf("%f", modelConfig.TopP),
		)
	}

	if modelConfig.TopK != 0 {
		args = append(args,
			"--top-k",
			fmt.Sprintf("%d", modelConfig.TopK),
		)
	}

	if modelConfig.RepeatPenalty != 0 {
		args = append(args,
			"--repeat-penalty",
			fmt.Sprintf("%f", modelConfig.RepeatPenalty),
		)
	}

	cmd := exec.Command(
		"llama-server",
		args...,
	)

	if env.DEBUG {
		// share stdout, stderr
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	llamaServer := &LlamaServer{
		ModelConfig: modelConfig,
		cmd:         cmd,
		port:        port,
	}

	var healthcheckError error
	for attempt := range 20 {
		time.Sleep(time.Millisecond * 50)
		fmt.Printf("check if llama-server is ready, attempt %d\n", attempt)

		healthcheckError = llamaServer.HealthCheck()
		if healthcheckError == nil {
			return llamaServer, nil
		}
	}

	llamaServer.Dispose()
	return nil, errors.Join(errors.New("llama-server failed to start"), healthcheckError)
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}
