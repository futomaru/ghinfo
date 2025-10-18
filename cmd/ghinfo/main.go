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

func main() {
	// コマンドライン引数の定義。
	var (
		timeout = flag.Duration("timeout", 5*time.Second, "request timeout(e.g. 3s, 1m)")
		asJSON  = flag.Bool("json", false, "print raw JSON (for piping)")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <owner>/<repo>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Please specify a repository in the form owner/repo.")
		flag.Usage()
		os.Exit(2)
	}
	repoID := flag.Arg(0)

	token := os.Getenv("GITHUB_TOKEN")

	client := githubapi.NewClient(*timeout, token)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	info, err := client.GetRepository(ctx, repoID)
	if err != nil {
		handleError(err)
	}

	if *asJSON {
		if err := outputJSON(info); err != nil {
			handleError(err)
		}
		return
	}

	if err := outputTable(info); err != nil {
		handleError(err)
	}
}

// handleError はエラーを整形して標準エラー出力へ表示し、終了ステータスを1に設定する。
func handleError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// outputJSON は取得した情報を整形した JSON 文字列で出力する。
func outputJSON(info githubapi.RepositoryInfo) error {
	// JSON を読みやすく整形するためにインデント付きでエンコードする。
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("JSONの整形に失敗しました: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputTable はタブ区切りを利用して情報を表形式で表示する。
func outputTable(info githubapi.RepositoryInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	// 表の各行を整形して出力。読みやすさを優先して日本語コメントを付与している。
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
