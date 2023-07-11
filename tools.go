package crawl

import (
	"archive/zip"
	"encoding/csv"
	"io"
	"net/http"
	"os"

	"github.com/17e10/go-httpb"
	"github.com/17e10/go-notifyb"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

const SkipAll notifyb.Notify = "skip all"

// Download は指定された URL からファイルを取得し 一時ファイルに保存します.
func Download(cl *Client, url string) (tmpname string, err error) {
	// GET でファイルを取得する
	resp, err := cl.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", httpb.ErrStatus(resp)
	}

	// 一時ファイルに保存する
	tmp, err := os.CreateTemp("", "crawl-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err = io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// ReadZipFiler は ScanZip の結果を受け取るインターフェイスです.
type ReadZipFiler interface {
	ReadFile(file *zip.File) error
	Done() error
}

// ReadZipFilerFunc は通常の関数を ReadZipFiler に変換するアダプタです.
type ReadZipFilerFunc func(*zip.File) error

// ReadFile は fn(file) を呼び出します.
func (fn ReadZipFilerFunc) ReadFile(file *zip.File) error {
	return fn(file)
}

// Done は fn(nil) を呼び出します.
func (fn ReadZipFilerFunc) Done() error {
	return fn(nil)
}

// ScanZip は Zip ファイルの内容をスキャンします.
//
// ファイルを見つける毎に filer.ReadFile を呼び出します.
// 全てのファイルをスキャンし終わると filer.Done を呼び出します.
//
// filer.ReadFile が SkipAll を返すと以降のスキャンをスキップします.
// この場合 filer.Done を呼び出しません.
func ScanZip(name string, filer ReadZipFiler) error {
	// zip ファイルを読み込む
	r, err := zip.OpenReader(name)
	if err != nil {
		return err
	}
	defer r.Close()

	// ファイルを展開する
	for _, f := range r.File {
		if err = filer.ReadFile(f); err == SkipAll {
			return nil
		} else if err != nil {
			return err
		}
	}
	return filer.Done()
}

// ReadCsvRower は ScanCsv の結果を受け取るインターフェイスです.
type ReadCsvRower interface {
	ReadRow(i int, row []string) error
	Done() error
}

// ReadCsvRowerFunc は通常の関数を ReadCsvRower に変換するアダプタです.
type ReadCsvRowerFunc func(int, []string) error

// ReadRow は fn(i, row) を呼び出します.
func (fn ReadCsvRowerFunc) ReadRow(i int, row []string) error {
	return fn(i, row)
}

// Done は fn(-1, nil) を呼び出します.
func (fn ReadCsvRowerFunc) Done() error {
	return fn(-1, nil)
}

// ScanCsv は CSV を読み込んでレコードに分解します.
//
// CSV の各行毎にカラムを []string に分解し rower.ReadRow を呼び出します.
// 全ての行をスキャンし終わると rower.Done を呼び出します.
//
// rower.ReadRow が SkipAll を返すと以降のスキャンをスキップします.
// この場合 rower.Done を呼び出しません.
//
// ScanCsv は golang.org/x/text/transform を使用して文字コードの変換をサポートします.
// enc に japanese.ShiftJIS などを渡すと指定したエンコーディングで変換します.
// enc に nil を渡すと変換せず UTF-8 として処理します.
func ScanCsv(r io.Reader, enc encoding.Encoding, rower ReadCsvRower) error {
	var (
		err error
		rec []string
	)
	if enc != nil {
		r = transform.NewReader(r, enc.NewDecoder())
	}
	csvr := csv.NewReader(r)
	for i := 0; ; i++ {
		if rec, err = csvr.Read(); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if err = rower.ReadRow(i, rec); err == SkipAll {
			return nil
		} else if err != nil {
			return err
		}
	}
	return rower.Done()
}
