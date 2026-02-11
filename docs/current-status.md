# ChatterBox 現状サマリー（2026-02-11）

## 概要
本ドキュメントは、直近のセキュリティ/安定性改善後の実装状態をまとめたものです。

## 直近で対応済みの課題
- 認証Cookieの改ざん耐性を追加
- 認証Cookieに `HttpOnly` / `SameSite` を設定
- 認証失敗時のCookieクリアを実装
- WebSocket処理での `log.Fatalln` を除去（単一リクエストでプロセス停止しない）
- `/upload` と `/uploader` を認証必須化
- アップロード時の `userid` をクライアント入力からサーバー側認証情報に変更
- OAuth `state` をランダム生成し、コールバック時に検証
- WebSocket `Origin` を同一ホスト/同一スキームに制限
- `room.Stop()` の `nil` チャネル問題を解消
- アバターアップロードに5MiB上限を導入
- Passkey登録を「開始時保存」から「完了時保存」に変更（仮登録残骸を抑制）

## 直近コミット
- `4a48a49` Harden auth cookie and protect upload endpoints
- `38b37fe` Add OAuth state cookie validation
- `62af929` Fix remaining security and lifecycle issues

## テスト状態
- 実行コマンド: `go test ./...`
- 結果: 全パッケージ成功

## 既知の注意点
- `AUTH_SECRET` 未設定時は起動ごとに一時鍵を生成するため、再起動後に既存ログインCookieは無効化されます。
- 本番運用では固定の `AUTH_SECRET` を環境変数で設定してください。

## 次に取り組む候補
- OAuthリフレッシュトークンの扱い見直し（`AccessTypeOffline` の要否確認）
- `secret.json` 依存の廃止（環境変数/シークレットマネージャへ移行）
- 永続ストレージ導入時のPasskeyデータモデル確定
