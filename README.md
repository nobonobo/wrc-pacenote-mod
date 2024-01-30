# wrc-pacenote-mod

「EA Sports WRC」のPC版向け、日本語コドライバーMOD

## あらかじめ必要なもの

- [最新リリース](https://github.com/nobonobo/wrc-pacenote-mod/releases/latest)から「wrc-pacenote-mod-####.zip」ファイルをダウンロード
- 「EA Sports WRC」のテレメトリ出力を有効に設定

## 展開と実行

- ダウンロードしたZIPファイルを「すべてを展開」します
- そのフォルダ内のwrc-pacenote-mod.exeを実行します

## 実行オプション

テレメトリパケットを受けるポートの指定
```
wrc-pacenote-mod -listen 127.0.0.1:20777
```

ほかにテレメトリパケットを必要とする機材やソフトウェアがあるなら転送を指定
```
wrc-pacenote-mod -forward 127.0.0.1:20778
```

テレメトリや音声、編集データ、ペースノートを保存するフォルダを指定。
デフォルトはEA Sports WRCのセーブデータの隣
（One Driveでドキュメントの場所が変更になってる場合などは注意してください）
```
wrc-pacenote-mod -log-dir ログ保管フォルダパス
```

## ログフォルダの構造

```
+-- WRC (EA Sports WRCのセーブデータフォルダ)
  +-- pacenotes/ (ログフォルダ)
    +-- ##.ロケーション名1
    | +-- 01.ステージ名1
    | +-- 02.ステージ名2
    | .
    | .
    | .
    | +-- 12.ステージ名12
    +-- ##.ロケーション名2
    | +-- 01.ステージ名1
    | +-- 02.ステージ名2
    | .
    | .
    | .
    | +-- 12.ステージ名12
    .
    .
    .
    +-- ##.ロケーション名##
    | +-- capture.wav (キャプチャ音声)
    | +-- telemetry.log (座標ログ)
    | +-- regeions.log （編集マーキングデータ）
    | +-- pacenote.log （生成ペースノート）
    +-- dictionary.json （発声単語辞書）
```

## 利用方法

1. ペースノートを自作したいステージを標準コドライバー音声ONで完走する
2. http://127.0.0.1:8080 を開くとステージ選択画面になります
3. 記録のあるステージを選ぶと編集画面になります
4. 読み上げ音声波形をマウスドラッグで選択、その内容をテキストで入力します
5. 一通り入力し、必要なら文言を追加して「Save」ボタンで保存します
6. 該当ステージを標準コドライバー音声OFFで走りこみます

## 編集画面の操作

- 区間は区間のないところをマウスドラッグで追加できます
- 区間をドラッグすると移動やリサイズができます
- unknown表記をクリックするとテキスト編集
- 区間をダブルクリックするとその区間の音声を再生
- Loopチェック有効なら区間内でリピート再生します
- 区間をクリックしたあとDeleteキーで区間の削除
- 区間がオーバーラップしてもあくまでペースノートが発火するのは区間の開始点です
- 音声再生に合わせて下部の地図上に自車位置が出ますのでペースノートを補完する際の参考に
- 地図はマウスホイールで拡大縮小、ドラッグで移動できます
- 「Save」ボタンで保存さえすれば後で編集は再開できます

## 既知の問題

- 小さすぎる区間ができてしまった場合はZoomで広げて操作してください
- 編集ペースノートがまだ保存されていないステージは自動的に記録を残すモードになります
- 記録を残す際に複数回行うと記録ファイルが連番の別ファイルに記録されます（過去の記録を消しません）
- なので、一番有効な記録を選んで後置の番号を取り除く作業が必要です（このあたりのUIはまだ未実装）
- テキストにはdictionary.jsonにある単語を指定する必要がある（動的単語でTTSやっているとクラッシュする問題がある）
- フィクション系ステージの一部でまだステージ長を確認できていないところがある

## 変更履歴

- base.jsonはテンプレート扱いとし、「ログルート/dictionary.json」を単語辞書扱いにした
- MyDucumentsフォルダの取得方法をWindowsAPI経由に[変更](https://zenn.dev/link/comments/0c61eaec7989e8)する
- onnxruntime.dll名称コンフリクト問題は[DLLパス設定](https://zenn.dev/link/comments/313573ed05b8b5)にて解決
- 以上によりカレントフォルダ縛りは解消された

## 改良予定（TODO）

- フィクション系ステージの一部未確認のステージ長を確認する
- ステージ選択に履歴と作成日時も表示してその選択でペースノートの元記録を選ぶ
- pacenote.logをパースした時に未知の単語があれば、あらかじめCreateAudioQueryしておく（動的な単語利用でクラッシュする問題の回避）
