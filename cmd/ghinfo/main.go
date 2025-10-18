package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"ghinfo/internal/githubapi"
)

// cliOptions は CLI で受け取った設定値をまとめる。
type cliOptions struct {
	repo    string
	timeout time.Duration
	asJSON  bool
	token   string
}

func main() {
	opts, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := run(opts); err != nil {
		handleError(err)
	}
}

// parseFlags はフラグ値と引数を解析して cliOptions を返す。
func parseFlags() (cliOptions, error) {
	timeout := flag.Duration("timeout", 5*time.Second, "request timeout (e.g. 3s, 1m)")
	asJSON := flag.Bool("json", false, "print JSON output")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <owner>/<repo>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return cliOptions{}, fmt.Errorf("please specify a repository in the form owner/repo")
	}

	return cliOptions{
		repo:    flag.Arg(0),
		timeout: *timeout,
		asJSON:  *asJSON,
		token:   os.Getenv("GITHUB_TOKEN"),
	}, nil
}

// run はクライアント生成から出力までの流れを担当する。
func run(opts cliOptions) error {
	client := githubapi.NewClient(opts.timeout, opts.token)

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	info, err := client.GetRepositoryByFullName(ctx, opts.repo)
	if err != nil {
		return err
	}

	if opts.asJSON {
		return outputJSON(info)
	}

	return outputTable(info)
}

// handleError はエラーを表示して終了する。
func handleError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// outputJSON はリポジトリ情報を JSON 形式で表示する。
func outputJSON(info githubapi.RepositoryInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("JSONの整形に失敗しました: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// outputTable はリポジトリ情報を表形式で表示する。
func outputTable(info githubapi.RepositoryInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(w, "Repository:\t%s\n", info.FullName)
	fmt.Fprintf(w, "URL:\t%s\n", info.HTMLURL)
	fmt.Fprintf(w, "Description:\t%s\n", info.Description)
	fmt.Fprintf(w, "Language:\t%s\n", info.Language)
	fmt.Fprintf(w, "License:\t%s\n", info.License)
	fmt.Fprintf(w, "Stars:\t%d\n", info.Stars)
	fmt.Fprintf(w, "Forks:\t%d\n", info.Forks)
	fmt.Fprintf(w, "Watchers:\t%d\n", info.Watchers)
	fmt.Fprintf(w, "Open Issues:\t%d\n", info.OpenIssues)
	fmt.Fprintf(w, "Archived:\t%t\n", info.Archived)
	fmt.Fprintf(w, "Disabled:\t%t\n", info.Disabled)
	fmt.Fprintf(w, "Updated:\t%s\n", info.UpdatedAt.Local().Format(time.RFC1123))
	fmt.Fprintf(w, "Pushed:\t%s\n", info.PushedAt.Local().Format(time.RFC1123))

	if err := w.Flush(); err != nil {
		return fmt.Errorf("表形式出力の書き込みに失敗しました: %w", err)
	}
	return nil
}
