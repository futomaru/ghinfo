# ghinfo

## 概要
`ghinfo` は指定した GitHub リポジトリの主な指標（スター数、フォーク数、最終更新、ライセンスなど）を表示する Go 製の小さな CLI ツールです。

> [!NOTE]
> これは **個人の学習のため** に作られたものです

## 使い方

```bash
ghinfo [flags] <owner>/<repo>
```

- `-timeout duration` リクエストのタイムアウト（既定値 `5s`）
- `-json` JSON 形式で出力

認証付きで利用したい場合は、環境変数 `GITHUB_TOKEN` を設定してください。

## ビルド

```bash
go build ./cmd/ghinfo
```
