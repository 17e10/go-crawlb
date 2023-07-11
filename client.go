package crawl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/17e10/go-crawlb/cache"
	"github.com/17e10/go-crawlb/mutex"
)

var (
	errNotStartedTx = errors.New("not started transaction")
)

// Client はキャッシュ機構とアクセス間隔制御機構を持つ HTTP クライアントです.
//
// Client はサーバに負荷を掛けないよう指定した間隔を空けてアクセスします.
//
// Client が持つキャッシュ機構はキャッシュディレクトリに
// トランザクション単位でアクセス結果をファイル保存します.
// トランザクションは世代管理していて複数世代を保持することができます.
// この仕組みによって障害発生時を再現したり サーバに負担を掛けずに開発・テストができます.
// トランザクションは最大世代数を超えると自動的に破棄されます.
type Client struct {
	ctx   context.Context
	mu    mutex.Mutex
	cache *cache.Cache
	tx    *cache.Tx
}

// NewClient は新しい Client を作成します.
//
// サーバへのアクセス間隔は d で指定します.
// キャッシュ機構のディレクトリやトランザクションの最大世代数はそれぞれ
// cacheDir, numTx で指定します.
func NewClient(ctx context.Context, d time.Duration, cacheDir string, numTx int) (*Client, error) {
	cache, err := cache.New(cacheDir, numTx)
	if err != nil {
		return nil, err
	}
	return &Client{
		ctx:   ctx,
		mu:    *mutex.New(d),
		cache: cache,
	}, nil
}

// NewTransaction は新しいトランザクションを開始し世代を切り替えます.
func (cl *Client) NewTransaction() error {
	tx, err := cl.cache.NewTransaction()
	if err != nil {
		return err
	}
	cl.tx = tx
	return nil
}

// LastTransaction は前回のトランザクションを再開します.
func (cl *Client) LastTransaction() error {
	tx, err := cl.cache.GetLastTransaction()
	if err != nil {
		return err
	}
	cl.tx = tx
	return nil
}

// SetTransaction は指定したトランザクションを使用します.
func (cl *Client) SetTransaction(name string) error {
	tx, err := cl.cache.GetTransaction(name)
	if err != nil {
		return err
	}
	cl.tx = tx
	return nil
}

// Do は http.Request を送信し http.Response を返します.
// もしトランザクションにキャッシュがあれば キャッシュされた結果を返します.
func (cl *Client) Do(req *http.Request) (*http.Response, error) {
	if cl.tx == nil {
		return nil, errNotStartedTx
	}

	cf, err := cl.tx.NewFile(req)
	if err != nil {
		return nil, err
	}
	if err = cl.fetchAndStore(cf, req); err != nil {
		return nil, err
	}
	return cf.Load()
}

// fetchAndStore は実際に http.Request を送信し http.Response をキャッシュを保存します.
func (cl *Client) fetchAndStore(cf *cache.File, req *http.Request) error {
	var (
		resp *http.Response
		err  error
	)

	if cf.IsExists() {
		return nil
	}

	if err = cl.mu.Lock(cl.ctx); err != nil {
		return err
	}
	defer cl.mu.Unlock()

	if req.Method == http.MethodGet && req.URL.Scheme == "file" {
		resp, err = fileResponse(path.Join("/", req.URL.Host, req.URL.Path))
	} else {
		resp, err = http.DefaultClient.Do(req)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return cf.Store(resp)
}

// Get は指定された URL に対して GET を発行します.
func (cl *Client) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequestWithContext(cl.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return cl.Do(req)
}

// Head は指定された URL に対して HEAD を発行します.
func (cl *Client) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequestWithContext(cl.ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return cl.Do(req)
}

// Post は指定された URL に対して POST を発行します.
//
// body をフォーム形式や JSON 形式で送信する場合
// それぞれ PostForm, PostJson を利用するとより簡単です.
func (cl *Client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequestWithContext(cl.ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return cl.Do(req)
}

// PostForm はペイロードをフォーム形式で POST を発行します.
func (cl *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	body := strings.NewReader(data.Encode())
	return cl.Post(url, "application/x-www-form-urlencoded", body)
}

// PostJson はペイロードを JSON 形式で POST を発行します.
func (cl *Client) PostJson(url string, data any) (resp *http.Response, err error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(b)
	return cl.Post(url, "application/json", body)
}

// fileResponse はローカルファイルを http.Rresponse で返します.
func fileResponse(filepath string) (resp *http.Response, err error) {
	gmt, err := time.LoadLocation("GMT")
	if err != nil {
		return nil, err
	}

	resp = &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	resp.Header = make(http.Header)
	resp.Header.Set("Content-Type", "application/zip")
	resp.Header.Set("X-Content-Type-Options", "nosniff")

	resp.Header.Set("Date", time.Now().In(gmt).Format(time.RFC1123))

	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	resp.ContentLength = stat.Size()
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	resp.Header.Set("Last-Modified", stat.ModTime().In(gmt).Format(time.RFC1123))

	resp.Body = f

	return resp, nil
}
