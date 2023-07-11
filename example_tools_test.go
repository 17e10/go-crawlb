package crawl_test

import (
	"archive/zip"
	"fmt"
	"os"

	crawl "github.com/17e10/go-crawlb"
	"golang.org/x/text/encoding/japanese"
)

var (
	cl *crawl.Client
)

func ExampleDownload() {
	tmpname, err := crawl.Download(cl, "https://example.com/dl/x")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(tmpname) // It is a temporary file, so it will be deleted when finished.

	f, err := os.Open(tmpname)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
}

type ZipReader struct{}

func (*ZipReader) ReadFile(file *zip.File) error {
	fmt.Println(file.Name)

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func (*ZipReader) Done() error {
	fmt.Println("done")
	return nil
}

func ExampleScanZip() {
	var zr ZipReader
	err := crawl.ScanZip("example.zip", &zr)
	if err != nil {
		fmt.Println(err)
	}
}

type CsvReader struct{}

func (*CsvReader) ReadRow(i int, row []string) error {
	fmt.Printf("len: %d\n", len(row))
	return nil
}

func (*CsvReader) Done() error {
	fmt.Println("done")
	return nil
}

func ExampleScanCsv() {
	f, err := os.Open("testfiles/sjis.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	var cr CsvReader
	err = crawl.ScanCsv(f, japanese.ShiftJIS, &cr)
	if err != nil {
		fmt.Println(err)
		return
	}
}
