package commands

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	updatepkg "github.com/creativeprojects/go-selfupdate/update"
	"github.com/google/go-github/v91/github"

	"github.com/xymaxim/ypb/internal/version"
)

const (
	repoOwner = "xymaxim"
	repoName  = "ypb"
)

type Update struct {
	Yes bool `short:"y" help:"Don't ask for confirmation."`
}

func (u *Update) Run() error {
	ctx := context.Background()

	gh, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("could not create github client: %w", err)
	}

	release, _, err := gh.Repositories.GetLatestRelease(ctx, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("could not fetch latest release: %w", err)
	}

	tag := release.GetTagName()

	if tag == version.GitVersion {
		fmt.Println("Already up to date:", version.GitVersion)
		return nil
	}

	fmt.Printf("New version available: %s\n", tag)
	fmt.Println(release.GetHTMLURL())

	if !u.Yes && !confirm("Continue?") {
		fmt.Println("Cancelled.")
		return nil
	}

	assetName := fmt.Sprintf("%s-%s-%s-%s.zip", repoName, tag, runtime.GOOS, runtime.GOARCH)

	var assetURL string
	for _, a := range release.Assets {
		if a.GetName() == assetName {
			assetURL = a.GetBrowserDownloadURL()
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no asset found matching %s", assetName)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable: %w", err)
	}

	fmt.Printf("Updating %s -> %s...\n", version.GitVersion, tag)

	if err := downloadAndApply(ctx, assetURL, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println("Complete!")
	return nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func downloadAndApply(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	binName := "ypb"
	if runtime.GOOS == "windows" {
		binName = "ypb.exe"
	}

	var binReader io.ReadCloser
	for _, f := range zr.File {
		if f.Name == binName {
			binReader, err = f.Open()
			if err != nil {
				return err
			}
			break
		}
	}
	if binReader == nil {
		return fmt.Errorf("binary %s not found in archive", binName)
	}
	defer binReader.Close()

	return updatepkg.Apply(binReader, updatepkg.Options{TargetPath: dest})
}
