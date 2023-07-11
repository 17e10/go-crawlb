package cache

import (
	"net/http"
	"testing"
)

func TestCache(t *testing.T) {
	cache, err := New("../../_var/cache", 3)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := cache.GetLastTransaction()
	// tx, err := cache.NewTransaction()
	if err != nil {
		t.Fatal(err)
	}

	/*
		req, err := http.NewRequest(
			http.MethodPost,
			"https://hlo.tohotheater.jp/net/ticket/066/TNPI2040J03.do",
			strings.NewReader("site_cd=066&jyoei_date=20230628&gekijyo_cd=0661&screen_cd=09&sakuhin_cd=022245&pf_no=7&fnc=1&pageid=2000J01&enter_kbn="),
		)
	*/
	req, err := http.NewRequest(
		http.MethodGet,
		"https://google.com/",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	cf, err := tx.NewFile(req)
	if err != nil {
		t.Fatal(err)
	}

	if !cf.IsExists() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if err = cf.Store(resp); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := cf.Load()
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Logf("X-Crawl-Cache: %s", resp.Header.Get("X-Crawl-Cache"))
}
