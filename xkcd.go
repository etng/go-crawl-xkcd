package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/etng/colly"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
)

type Comic struct {
	Url         string
	Title       string
	Description string
	PublishedAt string
	ImageUrl    string
}

func makeFilename(u *url.URL) string {
	return strings.TrimLeft(u.Path, "/")
}

func makeFilenameS(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		pu = nil
	}
	return makeFilename(pu)
}
func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
func main() {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36"
	outputFile := "xkcd.json"
	cacheDir := "/tmp/xkcd_cache"
	var limit int
	var logFile string
	flag.IntVar(&limit, "limit", 0, "limit result length")
	flag.StringVar(&logFile, "logto", "xkcd.log", "write logs to which file")
	flag.Parse()
	if logFile != "" {
		fLogFile, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer fLogFile.Close()

		log.SetOutput(fLogFile)
		fmt.Println("check log at", logFile)
	}
	comics := make([]Comic, 0, 200)

	c := colly.NewCollector()
	detail_collector := colly.NewCollector()
	downloader := colly.NewCollector()

	c.UserAgent = ua
	detail_collector.UserAgent = ua
	downloader.UserAgent = ua
	c.CacheDir = cacheDir
	detail_collector.CacheDir = cacheDir

	c.AllowedDomains = []string{"xkcd.com"}
	i := 0

	detail_collector.OnHTML("#middleContainer.box", func(e *colly.HTMLElement) {
		img_url := e.Request.AbsoluteURL(e.ChildAttr("img", "src"))
		comic := Comic{
			Title:       e.ChildText("#ctitle"),
			Description: e.ChildAttr("#comic img", "title"),
			Url:         e.Request.URL.String(),
			PublishedAt: e.Request.Ctx.Get("PublishedAt"),
			ImageUrl:    makeFilenameS(img_url),
		}
		comics = append(comics, comic)
		if PathExist(comic.ImageUrl) {
			log.Printf("image from %q already exists as %q", img_url, comic.ImageUrl)
			return
		}
		log.Printf("should download image from %q", img_url)
		downloader.Visit(img_url)
	})

	c.OnHTML(`#middleContainer.box a`, func(e *colly.HTMLElement) {
		url := e.Request.AbsoluteURL(e.Attr("href"))
		i += 1
		if limit > 0 && i > limit {
			return
		}
		log.Printf("visiting %q\n", url)
		ctx := colly.NewContext()
		ctx.Put("PublishedAt", e.Attr("title"))
		detail_collector.Scrape(url, "GET", 1, nil, ctx, nil)
	})

	downloader.OnResponse(func(r *colly.Response) {
		filename := makeFilename(r.Request.URL)
		log.Printf("saving file %q\n", filename)
		r.Save(filename)
	})

	start_url := "https://xkcd.com/archive/"
	log.Printf("begining with %q\n", start_url)
	c.Visit(start_url)

	jsonData, err := json.MarshalIndent(comics, "", "  ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(outputFile, jsonData, 0644)
	log.Printf("Scraping finished, check file %q for results\n", outputFile)
}
