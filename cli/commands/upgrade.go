package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
	upgradeReleaseURL = "https://api.github.com/repos/abdullahPrasetio/wapgo/releases/latest"
	upgradeInstallPkg = "github.com/abdullahPrasetio/wapgo/cli/wapgo"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func newUpgradeCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for a newer wapgo release and upgrade via go install",
		Long: `Fetch the latest release tag from GitHub and compare with the installed version.
If a newer version is available, run go install to upgrade.

Examples:
  wapgo upgrade           # check and upgrade if needed
  wapgo upgrade --check   # check only, do not install`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(checkOnly)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only report whether an update is available")
	return cmd
}

func runUpgrade(checkOnly bool) error {
	stTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	stOK    := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	stWarn  := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	stErr   := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	stDim   := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println()
	fmt.Println("  " + stTitle.Render("✦ wapgo upgrade"))
	fmt.Println()
	fmt.Printf("  %s  installed : %s\n", stDim.Render("→"), Version)

	rel, err := fetchLatestRelease()
	if err != nil {
		fmt.Printf("  %s\n\n", stErr.Render("✗ could not reach GitHub: "+err.Error()))
		return nil // offline is not a hard error
	}

	latestTag := rel.TagName // e.g. "v1.4.1"
	fmt.Printf("  %s  latest    : %s\n\n", stDim.Render("→"), latestTag)

	current := strings.TrimPrefix(Version, "v")
	latest  := strings.TrimPrefix(latestTag, "v")

	if Version == "dev" {
		fmt.Println("  " + stWarn.Render("⚠  dev build — cannot compare versions"))
		fmt.Printf("  %s  consider: go install %s@%s\n\n", stDim.Render("→"), upgradeInstallPkg, latestTag)
		return nil
	}

	if !semverGreater(latest, current) {
		fmt.Println("  " + stOK.Render("✓ already up to date"))
		fmt.Println()
		return nil
	}

	// Update available.
	fmt.Printf("  %s  update available: %s → %s\n\n",
		stWarn.Render("↑"), Version, latestTag)

	if checkOnly {
		fmt.Printf("  run %s to upgrade\n\n",
			stDim.Render("wapgo upgrade"))
		return nil
	}

	goExe := "go"
	if p, err := exec.LookPath("go"); err == nil {
		goExe = p
	}

	installRef := upgradeInstallPkg + "@" + latestTag
	fmt.Printf("  %s  running: go install %s\n\n", stDim.Render("→"), installRef)

	c := exec.Command(goExe, "install", installRef)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		fmt.Printf("\n  %s\n\n", stErr.Render("✗ go install failed — see output above"))
		return fmt.Errorf("go install failed")
	}

	fmt.Printf("\n  %s  run %s to confirm\n\n",
		stOK.Render("✓ upgraded to "+latestTag),
		stDim.Render("wapgo version"))
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, upgradeReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "wapgo-cli/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API status %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// semverGreater returns true when a > b using simple X.Y.Z integer comparison.
// Pre-release suffixes (e.g. "-rc1") are ignored.
func semverGreater(a, b string) bool {
	partsA := semverParts(a)
	partsB := semverParts(b)
	for i := range partsA {
		if partsA[i] != partsB[i] {
			return partsA[i] > partsB[i]
		}
	}
	return false
}

func semverParts(v string) [3]int {
	// strip pre-release suffix
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}
