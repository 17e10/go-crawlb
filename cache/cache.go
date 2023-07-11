// cache パッケージは http のキャッシュ機構を提供します.
package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/17e10/go-jsonb"
)

// ctlname はキャッシュの管理ファイル名です.
const ctlname = "cache.json"

var (
	errNoSuchTx = errors.New("no such transaction")
)

// Cache は http のキャッシュ機構を提供します.
type Cache struct {
	dir   string
	numTx int
	Trans []Tx `json:"transactions"`
}

// New は新しい Cache を作成します.
func New(dir string, numTx int) (*Cache, error) {
	c := &Cache{dir: dir, numTx: numTx}

	err := c.loadCtlFile()
	if err != nil && errors.Unwrap(err) != syscall.ENOENT {
		return nil, err
	}
	if err = c.discard(); err != nil {
		return nil, err
	}
	return c, nil
}

// 新しいトランザクションを作成します.
func (c *Cache) NewTransaction() (*Tx, error) {
	var zero Tx

	// 現在時刻からトランザクション名を作成する
	name := strconv.FormatInt(time.Now().UnixMilli(), 16)
	newtx, err := newTx(c.dir, name)
	if err != nil {
		return nil, err
	}

	c.Trans = append(c.Trans, zero)
	copy(c.Trans[1:], c.Trans[:len(c.Trans)])
	c.Trans[0] = *newtx

	if err = c.discard(); err != nil {
		return nil, err
	}
	if err = c.saveCtlFile(); err != nil {
		return nil, err
	}
	return newtx, nil
}

// GetLastTransaction は最後に作成されたトランザクションを返します.
func (c *Cache) GetLastTransaction() (*Tx, error) {
	if len(c.Trans) == 0 {
		c.NewTransaction()
	}
	return &c.Trans[0], nil
}

// GetTransaction は指定されたトランザクションを返します.
func (c *Cache) GetTransaction(name string) (*Tx, error) {
	for i, l := 0, len(c.Trans); i < l; i++ {
		if c.Trans[i].Name == name {
			return &c.Trans[i], nil
		}
	}
	return nil, fmt.Errorf("get transaction %q: %w", name, errNoSuchTx)
}

// discard はキャッシュ作成時に指定したトランザクション数を超えた
// 古いトランザクションを削除します.
func (c *Cache) discard() error {
	if len(c.Trans) <= c.numTx {
		return nil
	}
	for i, l := c.numTx, len(c.Trans); i < l; i++ {
		if err := os.RemoveAll(c.Trans[i].dir); err != nil {
			return err
		}
	}
	c.Trans = c.Trans[:c.numTx]
	return nil
}

// loadCtlFile は管理ファイルを読み込みます.
func (c *Cache) loadCtlFile() error {
	if err := jsonb.Load(path.Join(c.dir, ctlname), c); err != nil {
		return err
	}
	for i, l := 0, len(c.Trans); i < l; i++ {
		c.Trans[i].dir = path.Join(c.dir, c.Trans[i].Name)
	}
	return nil
}

// saveCtlFile は管理ファイルを保存します.
func (c *Cache) saveCtlFile() error {
	return jsonb.Save(path.Join(c.dir, ctlname), c)
}

// Tx はトランザクションを表します.
type Tx struct {
	Name     string `json:"name"`
	CreateAt string `json:"create_at"`
	dir      string
}

// newTx は新しい Tx を作成します.
func newTx(dir, name string) (*Tx, error) {
	dir = path.Join(dir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Tx{
		Name:     name,
		CreateAt: time.Now().Format("2006-01-02 15:04:05.000"),
		dir:      dir,
	}, nil
}

// NewFile は http.Request に対応した File を作成します.
func (tx *Tx) NewFile(req *http.Request) (*File, error) {
	creq, err := newCreq(req)
	if err != nil {
		return nil, err
	}
	pathname := path.Join(tx.dir, creq.ident())
	return &File{creq, pathname}, nil
}

// File は http.Request に対応したキャッシュファイルを表します.
type File struct { // TODO: rename 名称がしっくりこない
	creq     *cReq
	pathname string
}

// IsExists はキャッシュファイルがあるかを返します.
func (f *File) IsExists() bool {
	_, err := os.Stat(f.pathname)
	return err == nil
}

// Load はキャッシュファイルから http.Response を返します.
func (f *File) Load() (*http.Response, error) {
	var creq cReq
	var cres cRes

	r, err := os.Open(f.pathname)
	if err != nil {
		return nil, err
	}

	err = func(r *os.File) error {
		dec := json.NewDecoder(r)
		if err = dec.Decode(&creq); err != nil {
			return err
		}
		if err = dec.Decode(&cres); err != nil {
			return err
		}
		_, err = r.Seek(dec.InputOffset()+1, 0)
		return err
	}(r)
	if err != nil {
		r.Close()
		return nil, err
	}

	resp := cres.newResponse()
	u, _ := url.Parse(creq.Url)
	resp.Request = &http.Request{
		Method: creq.Method,
		URL:    u,
	}
	resp.Body = r

	return resp, nil
}

// Store はキャッシュファイルに http.Response の内容を保存します.
func (f *File) Store(resp *http.Response) error {
	cres := newCres(resp)

	w, err := os.Create(f.pathname)
	if err != nil {
		return err
	}
	defer w.Close()

	enc := json.NewEncoder(w)
	if err = enc.Encode(f.creq); err != nil {
		return err
	}
	if err = enc.Encode(cres); err != nil {
		return err
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// cReq はキャッシュファイルに格納するリクエスト情報を表します.
type cReq struct {
	Method  string
	Url     string
	Payload []byte
}

// newCreq は http.Request から新しい cReq を作成します.
func newCreq(req *http.Request) (*cReq, error) {
	payload, err := getPayload(req.GetBody)
	if err != nil {
		return nil, err
	}

	return &cReq{
		req.Method,
		req.URL.String(),
		payload,
	}, nil
}

// getPayload は http.Request からペイロードを取得します.
//
// ペイロードは Body からではなく GetBody から最大 256 bytes までを対象に取得します.
// POST PUT や PATCH などで JSON や Form 形式を渡す場合 http パッケージは GetBodyを準備しており
// 再利用可能な io.ReadCloser を返してくれるためです.
func getPayload(getBody func() (io.ReadCloser, error)) ([]byte, error) {
	const maxbytes = 256

	if getBody == nil {
		return nil, nil
	}

	r, err := getBody()
	if err != nil {
		return nil, err
	}
	w := &bytes.Buffer{}
	_, err = io.CopyN(w, r, maxbytes)
	if err != nil && err != io.EOF {
		return nil, err
	}
	r.Close()

	return w.Bytes(), nil
}

// ident は cReq の識別子を計算します.
//
// 識別子は Method, Url, Payload を元に MD5 チェックサムで計算されます.
func (cq *cReq) ident() string {
	b := bytes.NewBuffer(make([]byte, 0, 256))
	b.WriteString(cq.Method)
	b.WriteString(cq.Url)
	if cq.Payload != nil {
		b.Write(cq.Payload)
	}
	sum := md5.Sum(b.Bytes())
	return hex.EncodeToString(sum[:])
}

// cRes はキャッシュファイルに格納するレスポンス情報を表します.
type cRes struct {
	Status           string
	StatusCode       int
	Proto            string
	ProtoMajor       int
	ProtoMinor       int
	Header           http.Header
	ContentLength    int64
	TransferEncoding []string
	Uncompressed     bool
}

// newCres は http.Response から新しい cRes を作成します.
func newCres(resp *http.Response) *cRes {
	return &cRes{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMajor:       resp.ProtoMajor,
		ProtoMinor:       resp.ProtoMinor,
		Header:           resp.Header,
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Uncompressed:     resp.Uncompressed,
	}
}

// newResponse は cRes から http.Response を返します.
func (cs *cRes) newResponse() *http.Response {
	return &http.Response{
		Status:           cs.Status,
		StatusCode:       cs.StatusCode,
		Proto:            cs.Proto,
		ProtoMajor:       cs.ProtoMajor,
		ProtoMinor:       cs.ProtoMinor,
		Header:           cs.Header,
		ContentLength:    cs.ContentLength,
		TransferEncoding: cs.TransferEncoding,
		Uncompressed:     cs.Uncompressed,
	}
}
