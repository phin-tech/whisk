package controlauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	TokenFileName        = "control-token"
	AuthorizationHeader  = "Authorization"
	AccessTokenQueryName = "access_token"
)

var ErrNoToken = errors.New("whisk control auth token not found")

func StateDir() (string, error) {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "whisk"), nil
	}
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "whisk", "state"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "whisk"), nil
}

func TokenPath() (string, error) {
	stateDir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, TokenFileName), nil
}

func EnsureToken() (string, error) {
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	token, err := readTokenAt(path)
	if err == nil {
		if err := os.Chmod(path, 0o600); err != nil {
			return "", err
		}
		return token, nil
	}
	if err != nil && !errors.Is(err, ErrNoToken) {
		return "", err
	}
	return writeNewToken(path)
}

func ReadToken() (string, error) {
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	return readTokenAt(path)
}

func BearerHeader(token string) string {
	return "Bearer " + token
}

func BearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", false
	}
	return strings.TrimSpace(token), true
}

func TokensEqual(expected string, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	expectedHash := sha256.Sum256([]byte(expected))
	providedHash := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) == 1
}

func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

func readTokenAt(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNoToken
	}
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

func writeNewToken(path string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(dir, ".control-token-*")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return "", err
	}
	if _, err := temp.WriteString(token + "\n"); err != nil {
		_ = temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return "", err
	}
	cleanup = false
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
