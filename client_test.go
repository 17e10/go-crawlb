package crawl

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	const (
		cacheDir       = "../_var/cache"
		accessDuration = 2 * time.Second
		numTx          = 15
	)

	ctx := context.TODO()

	cl, err := NewClient(ctx, accessDuration, cacheDir, numTx)
	if err != nil {
		t.Fatal(err)
	}

	if err = cl.LastTransaction(); err != nil {
		t.Fatal(err)
	}

	resp, err := cl.Get("https://google.com/")
	if err != nil {
		t.Error("get google", err)
	}
	resp.Body.Close()

	vals := url.Values{}
	vals.Set("site_cd", "066")
	vals.Set("jyoei_date", "20230628")
	vals.Set("gekijyo_cd", "0661")
	vals.Set("screen_cd", "09")
	vals.Set("sakuhin_cd", "022245")
	vals.Set("pf_no", "7")
	vals.Set("fnc", "1")
	vals.Set("pageid", "2000J01")
	vals.Set("enter_kbn", "")
	resp, err = cl.PostForm("https://hlo.tohotheater.jp/net/ticket/066/TNPI2040J03.do", vals)
	if err != nil {
		t.Error("post tohotheater", err)
	}
	resp.Body.Close()
}
