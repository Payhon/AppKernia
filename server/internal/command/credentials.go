package command

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
)

type clientCredentials struct {
	Server       string `json:"server"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func authCommand(program string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s auth configure --server URL --client-id ID [--secret-stdin] [--credentials-file FILE]\n", program)
		return nil
	}
	if len(args) == 0 || args[0] != "configure" {
		return &UsageError{Message: fmt.Sprintf("usage: %s auth configure --server URL --client-id ID [--secret-stdin] [--credentials-file FILE]", program)}
	}
	flags := flag.NewFlagSet("auth configure", flag.ContinueOnError)
	server := flags.String("server", strings.TrimSpace(os.Getenv("AK_SERVER_URL")), "AppKernia server origin")
	clientID := flags.String("client-id", strings.TrimSpace(os.Getenv("AK_CLIENT_ID")), "API Client ID")
	secretStdin := flags.Bool("secret-stdin", false, "read the client secret from standard input")
	credentialsFile := flags.String("credentials-file", "", "credentials file path")
	if err := parseCommandFlags(flags, args[1:], fmt.Sprintf("usage: %s auth configure --server URL --client-id ID [--secret-stdin] [--credentials-file FILE]", program)); err != nil {
		return err
	}
	secret, err := readClientSecret(*secretStdin)
	if err != nil {
		return err
	}
	credentials := clientCredentials{Server: *server, ClientID: *clientID, ClientSecret: secret}
	if err = credentials.validate(); err != nil {
		return err
	}
	path, err := credentialsPath(*credentialsFile)
	if err != nil {
		return err
	}
	if err = writeCredentials(path, credentials); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"client_id": credentials.ClientID, "credentials_file": path, "server": credentials.Server})
}

func readClientSecret(fromStdin bool) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Client secret: ")
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read client secret: %w", err)
		}
		return strings.TrimSpace(string(value)), nil
	}
	if !fromStdin {
		return "", errors.New("standard input is not a terminal; pass --secret-stdin to read the client secret")
	}
	scanner := bufio.NewScanner(io.LimitReader(os.Stdin, 16*1024))
	if !scanner.Scan() {
		return "", fmt.Errorf("read client secret from stdin: %w", scanner.Err())
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func defaultCredentialsPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve credentials directory: %w", err)
	}
	return filepath.Join(directory, "AppKernia", "credentials.json"), nil
}

func credentialsPath(value string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("AK_CREDENTIALS_FILE"))
	}
	if path == "" {
		var err error
		path, err = defaultCredentialsPath()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve credentials file: %w", err)
	}
	return abs, nil
}

func writeCredentials(path string, credentials clientCredentials) error {
	content, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	content = append(content, '\n')
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err = os.Chmod(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("secure credentials directory: %w", err)
		}
	}
	if info, statErr := os.Lstat(path); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
		return errors.New("credentials path must be a regular file")
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect credentials file: %w", statErr)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open credentials file: %w", err)
	}
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(content)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	return nil
}

func loadCredentials(pathOverride, serverOverride string) (clientCredentials, error) {
	var credentials clientCredentials
	path, err := credentialsPath(pathOverride)
	if err != nil {
		return credentials, err
	}
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		if runtime.GOOS != "windows" {
			info, statErr := file.Stat()
			if statErr != nil {
				return credentials, fmt.Errorf("inspect credentials file: %w", statErr)
			}
			if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
				return credentials, errors.New("credentials file must be a regular owner-only file")
			}
		}
		decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&credentials); err != nil {
			return credentials, fmt.Errorf("decode credentials file: %w", err)
		}
		var extra any
		if err = decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return credentials, errors.New("credentials file must contain one JSON object")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return credentials, fmt.Errorf("read credentials file: %w", err)
	}
	if value := strings.TrimSpace(os.Getenv("AK_SERVER_URL")); value != "" {
		credentials.Server = value
	}
	if value := strings.TrimSpace(os.Getenv("AK_CLIENT_ID")); value != "" {
		credentials.ClientID = value
	}
	if value := strings.TrimSpace(os.Getenv("AK_CLIENT_SECRET")); value != "" {
		credentials.ClientSecret = value
	}
	if value := strings.TrimSpace(serverOverride); value != "" {
		credentials.Server = value
	}
	if err = credentials.validate(); err != nil {
		return credentials, fmt.Errorf("configure credentials with `akone auth configure` or AK_SERVER_URL, AK_CLIENT_ID, and AK_CLIENT_SECRET: %w", err)
	}
	return credentials, nil
}

func (credentials *clientCredentials) validate() error {
	credentials.Server = strings.TrimRight(strings.TrimSpace(credentials.Server), "/")
	credentials.ClientID = strings.TrimSpace(credentials.ClientID)
	credentials.ClientSecret = strings.TrimSpace(credentials.ClientSecret)
	parsed, err := url.Parse(credentials.Server)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("server must be an HTTP(S) origin without credentials, path, query, or fragment")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1")) {
		return errors.New("server must use HTTPS except on localhost")
	}
	if !strings.HasPrefix(credentials.ClientID, "ak_") {
		return errors.New("client ID must start with ak_")
	}
	if !strings.HasPrefix(credentials.ClientSecret, "aks_") || len(credentials.ClientSecret) < 32 {
		return errors.New("client secret is invalid")
	}
	return nil
}
