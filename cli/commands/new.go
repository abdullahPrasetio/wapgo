package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

// Styles for the post-scaffold summary.
var (
	stTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	stOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	stKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stVal     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	stCmd     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	stBoxLine = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func newNewCmd() *cobra.Command {
	var (
		module                            string
		db                                string
		apm                               string
		redis, kafka, rabbit              bool
		email, firebase, googleAuth       bool
		yes                               bool
	)

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Scaffold a new wapgo project (interactive)",
		Long: `Create a new wapgo project.

Run without flags for an interactive wizard:
  wapgo new

Or pass everything up-front for non-interactive / CI use:
  wapgo new shop --module github.com/me/shop --db mysql --redis --apm otel --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := ""
			if len(args) == 1 {
				projectName = args[0]
			}

			interactive := !yes && isatty.IsTerminal(os.Stdin.Fd())

			if interactive {
				if err := runWizard(&projectName, &module, &db, &apm, &redis, &kafka, &rabbit, &email, &firebase, &googleAuth); err != nil {
					return err
				}
			} else {
				// Non-interactive: apply defaults.
				if projectName == "" {
					return errors.New("project name is required in non-interactive mode (pass it as the first argument)")
				}
				if db == "" {
					db = "postgres"
				}
				if apm == "" {
					apm = "none"
				}
			}

			if module == "" {
				module = "github.com/example/" + strings.ReplaceAll(projectName, "_", "-")
			}

			targetDir := filepath.Join(".", projectName)
			if _, err := os.Stat(targetDir); err == nil {
				return fmt.Errorf("directory '%s' already exists", projectName)
			}

			opts := generator.ScaffoldOptions{
				ProjectName: projectName,
				Module:      module,
				DB:          db,
				APM:         apm,
				Redis:       redis,
				Kafka:       kafka,
				RabbitMQ:    rabbit,
				Email:       email,
				Firebase:    firebase,
				GoogleAuth:  googleAuth,
			}

			if err := generator.Scaffold(generator.TemplateFS, opts, targetDir); err != nil {
				return fmt.Errorf("scaffold failed: %w", err)
			}

			fmt.Println("  Running go mod tidy...")
			tidy := exec.Command("go", "mod", "tidy")
			tidy.Dir = targetDir
			tidy.Stdout = os.Stdout
			tidy.Stderr = os.Stderr
			if err := tidy.Run(); err != nil {
				fmt.Printf("  warning: go mod tidy failed: %v\n", err)
			}

			printSummary(opts)
			return nil
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module path (default: github.com/example/<project-name>)")
	cmd.Flags().StringVar(&db, "db", "", "Database driver: postgres | mysql")
	cmd.Flags().StringVar(&apm, "apm", "", "Observability provider: elastic_apm | otel | none")
	cmd.Flags().BoolVar(&redis, "redis", false, "Include Redis cache layer")
	cmd.Flags().BoolVar(&kafka, "kafka", false, "Include Kafka producer/consumer")
	cmd.Flags().BoolVar(&rabbit, "rabbitmq", false, "Include RabbitMQ publisher/consumer")
	cmd.Flags().BoolVar(&email, "email", false, "Include SMTP email add-on")
	cmd.Flags().BoolVar(&firebase, "firebase", false, "Include Firebase FCM push notification add-on")
	cmd.Flags().BoolVar(&googleAuth, "google-auth", false, "Include Google OAuth2 add-on")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Non-interactive: use flags/defaults, skip the wizard")

	return cmd
}

// runWizard drives the interactive huh form, mutating the passed-in values.
func runWizard(projectName, module, db, apm *string, redis, kafka, rabbit, email, firebase, googleAuth *bool) error {
	if *db == "" {
		*db = "postgres"
	}
	if *apm == "" {
		*apm = "none"
	}

	// Collect feature toggles into a multi-select slice, seeded from flags.
	features := []string{}
	if *redis {
		features = append(features, "redis")
	}
	if *kafka {
		features = append(features, "kafka")
	}
	if *rabbit {
		features = append(features, "rabbitmq")
	}
	if *email {
		features = append(features, "email")
	}
	if *firebase {
		features = append(features, "firebase")
	}
	if *googleAuth {
		features = append(features, "google-auth")
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Description("Used as the directory name and APP_NAME").
				Placeholder("my-service").
				Value(projectName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("project name cannot be empty")
					}
					if strings.ContainsAny(s, "/\\ ") {
						return errors.New("no slashes or spaces allowed")
					}
					return nil
				}),

			huh.NewInput().
				Title("Module path").
				Description("Go module path for go.mod (leave blank to auto-derive)").
				Placeholder("github.com/me/my-service").
				Value(module),

			huh.NewSelect[string]().
				Title("Database").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MySQL", "mysql"),
				).
				Value(db),

			huh.NewSelect[string]().
				Title("Observability / APM").
				Description("Tracing backend; choose None to disable").
				Options(
					huh.NewOption("Elastic APM", "elastic_apm"),
					huh.NewOption("OpenTelemetry", "otel"),
					huh.NewOption("None", "none"),
				).
				Value(apm),

			huh.NewMultiSelect[string]().
				Title("Optional features").
				Description("Space to toggle · Enter to confirm · can also be added later with `wapgo add`").
				Options(
					huh.NewOption("Redis cache", "redis"),
					huh.NewOption("Kafka producer/consumer", "kafka"),
					huh.NewOption("RabbitMQ publisher/consumer", "rabbitmq"),
					huh.NewOption("Email (SMTP mailer)", "email"),
					huh.NewOption("Firebase FCM push notification", "firebase"),
					huh.NewOption("Google OAuth2 login", "google-auth"),
				).
				Value(&features),
		),
	).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return err
	}

	*redis = contains(features, "redis")
	*kafka = contains(features, "kafka")
	*rabbit = contains(features, "rabbitmq")
	*email = contains(features, "email")
	*firebase = contains(features, "firebase")
	*googleAuth = contains(features, "google-auth")
	return nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func printSummary(opts generator.ScaffoldOptions) {
	check := stOK.Render("✓")

	feat := func(on bool) string {
		if on {
			return stOK.Render("enabled")
		}
		return stKey.Render("—")
	}

	rows := [][2]string{
		{"module", opts.Module},
		{"database", opts.DB},
		{"observability", opts.APM},
		{"redis", feat(opts.Redis)},
		{"kafka", feat(opts.Kafka)},
		{"rabbitmq", feat(opts.RabbitMQ)},
		{"email", feat(opts.Email)},
		{"firebase", feat(opts.Firebase)},
		{"google-auth", feat(opts.GoogleAuth)},
	}

	fmt.Println()
	fmt.Println("  " + stTitle.Render("✦ wapgo — project created") + "  " + check)
	fmt.Println()
	for _, r := range rows {
		fmt.Printf("    %s  %s\n", stKey.Render(pad(r[0], 14)), stVal.Render(r[1]))
	}
	fmt.Println()
	fmt.Println("  " + stBoxLine.Render("Next steps"))
	fmt.Printf("    %s\n", stCmd.Render("cd "+opts.ProjectName))
	fmt.Printf("    %s\n", stCmd.Render("cp .env.example .env"))
	fmt.Printf("    %s\n", stCmd.Render("make docker-up"))
	fmt.Printf("    %s\n", stCmd.Render("make run"))
	fmt.Println()
	fmt.Printf("  Add a domain:   %s\n", stCmd.Render("wapgo make:all product"))
	fmt.Printf("  Add a feature:  %s\n", stCmd.Render("wapgo add redis | kafka | rabbitmq | email | firebase | google-auth"))
	fmt.Println()
}

func pad(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}
