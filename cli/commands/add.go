package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

// feature describes an optional capability that can be added to an existing project.
type feature struct {
	name    string
	subdir  string   // skeleton subtree to copy
	summary string   // short description shown in help
	steps   []string // manual wiring instructions printed after copying
}

var features = map[string]feature{
	"redis": {
		name:    "redis",
		subdir:  "internal/repository/redis",
		summary: "Redis cache layer (RedisCacher implementing the domain Cacher)",
		steps: []string{
			"Wire it in cmd/api/main.go:",
			"    redisClient := newRedisClient(&cfg.Redis)",
			"    cache := redisrepo.NewRedisCacher(redisClient)",
			"Add REDIS_URL / REDIS_PASSWORD / REDIS_DB to your .env",
			"Add a redis service to docker-compose.yml (see the wapgo docs)",
		},
	},
	"kafka": {
		name:    "kafka",
		subdir:  "pkg/messaging/kafka",
		summary: "Kafka producer/consumer with health check",
		steps: []string{
			"Publish: producer := kafka.NewProducer(cfg.Kafka.Brokers, topic)",
			"Consume: kafka.NewConsumer(...).Start(ctx, handler)",
			"Add KAFKA_BROKERS / KAFKA_GROUP_ID to your .env",
			"Add kafka + zookeeper services to docker-compose.yml",
		},
	},
	"rabbitmq": {
		name:    "rabbitmq",
		subdir:  "pkg/messaging/rabbitmq",
		summary: "RabbitMQ publisher/consumer with health check",
		steps: []string{
			"Publish: pub := rabbitmq.NewPublisher(cfg.RabbitMQ.DSN, exchange)",
			"Subscribe: rabbitmq.NewConsumer(...).Subscribe(ctx, handler)",
			"Add RABBITMQ_DSN / RABBITMQ_EXCHANGE to your .env",
			"Add a rabbitmq service to docker-compose.yml",
		},
	},
	"email": {
		name:    "email",
		subdir:  "pkg/notification/email",
		summary: "SMTP mailer dengan OTel tracing dan journal integration",
		steps: []string{
			"Wire di cmd/api/main.go:",
			"    mailer := email.NewSMTPMailer(email.Config{Host: cfg.SMTP.Host, ...}, logger)",
			"Inject mailer ke usecase/handler yang butuh kirim email",
			"Tambah ke .env: SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM",
		},
	},
	"firebase": {
		name:    "firebase",
		subdir:  "pkg/notification/firebase",
		summary: "Firebase Cloud Messaging (FCM) push notification via FCM v1 HTTP API",
		steps: []string{
			"Wire di cmd/api/main.go:",
			"    pusher, err := firebase.NewFCMClient(os.Getenv(\"FIREBASE_CREDENTIALS_JSON\"), logger)",
			"Inject pusher ke usecase/handler yang butuh kirim push notification",
			"Tambah ke .env: FIREBASE_CREDENTIALS_JSON (isi JSON service account, lihat .env.example)",
		},
	},
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <feature>",
		Short: "Add an optional feature to an existing wapgo project",
		Long: `Add an optional feature to the project in the current directory.

Available features:
  redis      Redis cache layer
  kafka      Kafka producer/consumer
  rabbitmq   RabbitMQ publisher/consumer
  email      SMTP mailer
  firebase   Firebase FCM push notification

Example:
  wapgo add redis`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: featureNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args[0])
		},
	}
}

func runAdd(name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	f, ok := features[name]
	if !ok {
		return fmt.Errorf("unknown feature %q (available: %s)", name, strings.Join(featureNames(), ", "))
	}

	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	created, err := generator.AddFeatureFiles(generator.TemplateFS, f.subdir, module, ".")
	if err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}

	stOK := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	stKey := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))

	fmt.Println()
	fmt.Println("  " + stTitle.Render("✦ wapgo add "+name))
	fmt.Println()

	if len(created) == 0 {
		fmt.Println("  " + stKey.Render("Nothing to do — all files already exist."))
		fmt.Println()
		return nil
	}

	for _, c := range created {
		fmt.Printf("    %s %s\n", stOK.Render("created"), c)
	}

	fmt.Println()
	fmt.Println("  " + stTitle.Render("Next steps"))
	for _, s := range f.steps {
		fmt.Printf("    %s\n", s)
	}
	fmt.Println()
	return nil
}

func featureNames() []string {
	names := make([]string, 0, len(features))
	for k := range features {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
