# ghinfo

## 概要
`ghinfo` は `<owner>/<repo>` を指定して GitHub リポジトリの情報（スター数、フォーク数、更新日時、ライセンスなど）を取得・表示する CLI ツール。Go の標準ライブラリのみで構築する。

## 背景
学習のために、GoでAPI連携のCLIツールを作ってみる。
学習の目的は以下の2つ。
* Goに慣れる
* APIを理解する

## 機能要件
* `<owner>/<repo>` を引数に取り、以下の情報を取得・整形表示。

  * フルネーム、説明、URL、主要言語、ライセンス
  * Stars / Forks / Watchers / Open Issues
  * UpdatedAt / PushedAt
* オプション：

  * `-timeout <duration>`：リクエストのタイムアウト（デフォルト5s）
  * `-json`：JSON形式で出力
* 認証：環境変数 `GITHUB_TOKEN` による任意のトークン指定（レート制限緩和）

## CLI仕様

```
Usage: ghinfo [flags] <owner>/<repo>

Flags:
  -timeout duration   request timeout (e.g. 3s, 1m) [default: 5s]
  -json               print JSON format

Env:
  GITHUB_TOKEN        personal access token (optional)
```

## 出力例

### 表形式

```
Repository:   futomaru/ghinfo
URL:          https://github.com/futomaru/ghinfo
Description:  
Language:     Go
License:      
Stars:        0
Forks:        0
Watchers:     0
Open Issues:  0
Archived:     false
Disabled:     false
Updated:      Sat, 18 Oct 2025 23:36:43 JST
Pushed:       Sat, 18 Oct 2025 23:36:40 JST
```

### JSON形式

```json
{
  "FullName": "futomaru/ghinfo",
  "Description": "",
  "HTMLURL": "https://github.com/futomaru/ghinfo",
  "Language": "Go",
  "License": "",
  "Stars": 0,
  "Forks": 0,
  "Watchers": 0,
  "OpenIssues": 0,
  "Archived": false,
  "Disabled": false,
  "UpdatedAt": "2025-10-18T14:36:43Z",
  "PushedAt": "2025-10-18T14:36:40Z"
}
```

## アーキテクチャ

* **構成**：

  * `cmd/ghinfo/main.go`：CLI本体（引数解析・出力整形）
  * `internal/githubapi/github.go`：APIクライアント（HTTP, JSON, context）
* **通信**：`net/http` と `context.WithTimeout` によるタイムアウト制御。
* **出力**：`text/tabwriter` による表形式出力、または `-json` による JSON 出力。
* **設定**：`GITHUB_TOKEN` で認証、未設定でも匿名利用可能。
* **エラー処理**：HTTP ステータスとレスポンス本文を明確に整形して返却。
* **バージョン固定**：`X-GitHub-Api-Version: 2022-11-28` を送出。
