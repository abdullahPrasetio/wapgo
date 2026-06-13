package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

type workerData struct {
	Module   string
	Suffix   string // "worker" | "worker-order" | etc.
	Kafka    bool
	RabbitMQ bool
}

func newMakeWorkerCmd() *cobra.Command {
	var broker string
	cmd := &cobra.Command{
		Use:   "make:worker [name]",
		Short: "Generate a standalone background consumer entrypoint",
		Long: `Generate a worker binary that runs Kafka/RabbitMQ consumers as a process
separate from the API server. An optional name scopes the worker to a domain.

Output paths:
  (no name)          → cmd/worker/main.go          bin/worker
  make:worker order  → cmd/worker-order/main.go    bin/worker-order
  make:worker pay    → cmd/worker-pay/main.go      bin/worker-pay

Flags:
  --broker kafka      (default, or auto-detected from go.mod)
  --broker rabbitmq
  --broker both       (Kafka + RabbitMQ in one process)

Examples:
  wapgo make:worker
  wapgo make:worker order
  wapgo make:worker order --broker kafka
  wapgo make:worker notif --broker rabbitmq
  wapgo make:worker hub   --broker both`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runMakeWorker(name, broker)
		},
	}
	cmd.Flags().StringVar(&broker, "broker", "", "messaging broker: kafka | rabbitmq | both (auto-detected if omitted)")
	return cmd
}

func runMakeWorker(name, broker string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	if broker == "" {
		broker = detectWorkerBroker()
		if broker == "" {
			broker = "kafka"
		}
		fmt.Printf("  detected  broker=%s (use --broker to override)\n", broker)
	}

	data := workerData{
		Module:   module,
		Kafka:    broker == "kafka" || broker == "both",
		RabbitMQ: broker == "rabbitmq" || broker == "both",
	}

	if !data.Kafka && !data.RabbitMQ {
		return fmt.Errorf("unknown broker %q — choose kafka, rabbitmq, or both", broker)
	}

	// Build suffix and output dir from optional name
	suffix := "worker"
	outDir := "cmd/worker"
	if name != "" {
		slug := generator.NewNames(name).Snake
		suffix = "worker-" + slug
		outDir = "cmd/worker-" + slug
	}
	data.Suffix = suffix

	content, err := generator.DomainTemplateContent(generator.TemplateFS, "worker_main.go.tmpl")
	if err != nil {
		return err
	}

	out := filepath.Join(outDir, "main.go")
	if err := generator.Render(content, out, data); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)

	appendWorkerMakeTargets(suffix, outDir)
	return nil
}

// detectWorkerBroker reads go.mod to infer which broker packages are present.
func detectWorkerBroker() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	content := string(data)
	hasKafka := strings.Contains(content, "segmentio/kafka-go")
	hasRabbitMQ := strings.Contains(content, "rabbitmq/amqp091-go")
	switch {
	case hasKafka && hasRabbitMQ:
		return "both"
	case hasKafka:
		return "kafka"
	case hasRabbitMQ:
		return "rabbitmq"
	default:
		return ""
	}
}

// appendWorkerMakeTargets appends run-<suffix> and build-<suffix> to Makefile.
// Silently skips if no Makefile is found or targets already exist.
func appendWorkerMakeTargets(suffix, outDir string) {
	runTarget := "run-" + suffix
	buildTarget := "build-" + suffix
	binary := "bin/" + suffix

	targets := fmt.Sprintf(`
.PHONY: %s %s

%s:
	go run %s/main.go

%s:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o %s ./%s
`, runTarget, buildTarget,
		runTarget, outDir,
		buildTarget, binary, outDir)

	data, err := os.ReadFile("Makefile")
	if err != nil {
		return
	}
	if strings.Contains(string(data), runTarget) {
		return
	}
	f, err := os.OpenFile("Makefile", os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(targets) //nolint:errcheck
	fmt.Printf("  updated  Makefile (+%s, +%s)\n", runTarget, buildTarget)
}
