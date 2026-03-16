package cmd

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/ashrocket/bbcli/internal/config"
	"github.com/ashrocket/bbcli/internal/errors"
)

// Auth login flags.
var (
	authLoginToken string
)

// newAuthLoginCmd creates the `auth login` subcommand.
func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Bitbucket",
		Long: `Authenticate with Bitbucket using an app password or token.

In TTY mode, prompts for the token interactively (input is masked).
In non-TTY mode, the --token flag is required.

The token is validated by calling the Bitbucket /user endpoint.
On success, the token is stored in ~/.config/bbcli/credentials
with 0600 permissions.`,
		// Override PersistentPreRunE — auth login must work without
		// existing credentials (it is establishing them).
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: runAuthLogin,
	}

	cmd.Flags().StringVar(&authLoginToken, "token", "", "Authentication token (required in non-TTY mode)")

	return cmd
}

// runAuthLogin handles the `auth login` command.
// It skips the normal PersistentPreRunE auth chain because we're establishing
// credentials, not using existing ones.
func runAuthLogin(cmd *cobra.Command, args []string) error {
	token := authLoginToken

	if token == "" {
		// Check if stdin is a terminal.
		stat, _ := os.Stdin.Stat()
		isTTY := (stat.Mode() & os.ModeCharDevice) != 0
		if !isTTY {
			return errors.NewUsageError("--token flag is required in non-TTY mode")
		}
		// Prompt for token interactively.
		fmt.Fprint(os.Stderr, "Bitbucket token: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to read token: %v", err))
		}
		token = strings.TrimSpace(line)
	}

	if token == "" {
		return errors.NewUsageError("token cannot be empty")
	}

	// Validate the token by calling GET /user.
	tempAuth := &loginAuthenticator{token: token}
	client := api.New(tempAuth)
	raw, err := client.Request("GET", "/user", nil)
	if err != nil {
		return errors.NewAuthError(
			"token validation failed: "+err.Error(),
			"Ensure your token is a valid Bitbucket app password (format: user:app_password) or OAuth token",
		)
	}

	// Parse user info for the confirmation message.
	var user struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
		Nickname    string `json:"nickname"`
	}
	if jsonErr := json.Unmarshal(raw, &user); jsonErr != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to parse user response: %v", jsonErr))
	}

	username := user.Username
	if username == "" {
		username = user.Nickname
	}

	// Store the token.
	credDir := config.Dir()
	credPath := filepath.Join(credDir, "credentials")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to create config directory: %v", err))
	}
	if err := os.WriteFile(credPath, []byte(token+"\n"), 0600); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to write credentials: %v", err))
	}

	fmt.Fprintf(os.Stdout, "Authenticated as: %s (@%s)\n", user.DisplayName, username)
	return nil
}

// loginAuthenticator implements api.Authenticator for token validation during login.
type loginAuthenticator struct {
	token string
}

func (a *loginAuthenticator) Authenticate(req *http.Request) error {
	if strings.Contains(a.token, ":") {
		encoded := base64.StdEncoding.EncodeToString([]byte(a.token))
		req.Header.Set("Authorization", "Basic "+encoded)
	} else {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	return nil
}

func (a *loginAuthenticator) Source() string {
	return "login"
}

func (a *loginAuthenticator) Level() int {
	return 0
}
