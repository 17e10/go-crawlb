package crawl

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/text/encoding/japanese"
)

func TestDownload(t *testing.T) {
	const (
		cacheDir       = "../_var/cache"
		accessDuration = 2 * time.Second
		numTx          = 3
	)
	ctx := context.TODO()

	cl, err := NewClient(ctx, accessDuration, cacheDir, numTx)
	if err != nil {
		t.Fatal(err)
	}
	if err = cl.LastTransaction(); err != nil {
		t.Fatal(err)
	}

	tmpname, err := Download(cl, "https://www.post.japanpost.jp/zipcode/dl/kogaki/zip/ken_all.zip")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("tmpname:", tmpname)
	os.Remove(tmpname)
}

type tReadZipFilerEvent struct {
	events TestEvents
}

func (ev *tReadZipFilerEvent) ReadFile(file *zip.File) error {
	ev.events.Add(file.Name)
	return nil
}

func (ev *tReadZipFilerEvent) Done() error {
	ev.events.Add("done")
	return nil
}

func TestScanZip(t *testing.T) {
	var ev tReadZipFilerEvent
	err := ScanZip("testfiles/example.zip", &ev)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"ADD_2304.CSV",
		"done",
	}
	got := []string(ev.events)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", "test ScanZip", got, want)
	}
}

type tReadCsvRower struct {
	events TestEvents
}

func (ev *tReadCsvRower) ReadRow(i int, row []string) error {
	ev.events.Add(fmt.Sprintf("len: %d, %s", len(row), row[3]))
	return nil
}

func (ev *tReadCsvRower) Done() error {
	ev.events.Add("done")
	return nil
}

func TestScanCsvSjis(t *testing.T) {
	f, err := os.Open("testfiles/sjis.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var ev tReadCsvRower
	err = ScanCsv(f, japanese.ShiftJIS, &ev)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"len: 15, ﾐﾔｷﾞｹﾝ",
		"len: 15, ﾐﾔｷﾞｹﾝ",
		"len: 15, ﾆｲｶﾞﾀｹﾝ",
		"len: 15, ﾅｶﾞﾉｹﾝ",
		"len: 15, ﾋｮｳｺﾞｹﾝ",
		"len: 15, ﾔﾏｸﾞﾁｹﾝ",
		"done",
	}
	got := []string(ev.events)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", "test ScanCsv(sjis)", got, want)
	}
}

func TestScanCsvUtf8(t *testing.T) {
	f, err := os.Open("testfiles/utf8.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var ev tReadCsvRower
	err = ScanCsv(f, nil, &ev)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"len: 15, ﾐﾔｷﾞｹﾝ",
		"len: 15, ﾐﾔｷﾞｹﾝ",
		"len: 15, ﾅｶﾞﾉｹﾝ",
		"len: 15, ﾔﾏｸﾞﾁｹﾝ",
		"done",
	}
	got := []string(ev.events)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", "test ScanCsv(utf8)", got, want)
	}
}
