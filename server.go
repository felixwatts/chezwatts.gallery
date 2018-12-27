package main

import (
	"fmt"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"sync"
	"sort"
	"os"
	"encoding/csv"
	"strconv"
	"time"
)

const portHttp = 8081
const portHttps = 8443

const httpsRedirectRoot = "https://chezwatts.gallery:443"

const fileSystemRoot = "/var/www/chezwatts.gallery/"

const httpsCertificate = "/etc/letsencrypt/live/chezwatts.gallery/fullchain.pem"
const httpsPrivateKey = "/etc/letsencrypt/live/chezwatts.gallery/privkey.pem"

var templates = make(map[string]*template.Template)
var hitCountByPage = make(map[string]int)
var hitCountModifyLock = &sync.Mutex{}

func main() {

	defer saveStats()

	restoreStats()

	httpsMux := http.NewServeMux()

	httpsMux.HandleFunc("/favicon.ico", faviconHandler)
	httpsMux.HandleFunc("/", indexHandler)
	httpsMux.HandleFunc("/gallery/", galleryHandler)
	httpsMux.HandleFunc("/stats", statsHandler)
	httpsMux.Handle("/galleries/", http.StripPrefix("/galleries/", http.FileServer(http.Dir(fileSystemRoot + "galleries"))))
	httpsMux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(fileSystemRoot + "js"))))
	httpsMux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(fileSystemRoot + "css"))))

	httpMux := http.NewServeMux()

	httpMux.Handle("/.well-known/acme-challenge/", http.StripPrefix("/.well-known/acme-challenge/", http.FileServer(http.Dir(fileSystemRoot + ".well-known/acme-challenge"))))
	httpMux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(fileSystemRoot + "img"))))
	httpMux.HandleFunc("/", redirectToHttpsHandler)

	go http.ListenAndServe(":" + strconv.Itoa(portHttp), logAndDelegate(httpMux))
	log.Fatal(http.ListenAndServeTLS(":" + strconv.Itoa(portHttps), httpsCertificate, httpsPrivateKey, logAndDelegate(httpsMux)))
}

func init() {
	for _, tmpl := range []string{"index", "gallery", "stats"} {
		filename := fileSystemRoot + tmpl + ".html"
		t, err := template.ParseFiles(filename)
		if err != nil {
			panic(err)
		}

		templates[tmpl] = t
	}

	t, err := template.ParseFiles(fileSystemRoot + "stats.csv.tmpl")
	if err != nil {
		panic(err)
	}

	templates["stats_csv"] = t
}

func logAndDelegate(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		log.Println(time.Now(), r.Method, r.URL.Path, r.RemoteAddr, r.Referer(), r.UserAgent())
		handler.ServeHTTP(w, r)
	})
}

func redirectToHttpsHandler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, httpsRedirectRoot + r.RequestURI, http.StatusMovedPermanently)
}

func saveStats() {
	f, err := os.Create(fileSystemRoot + "stats.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	vm := getStatsPageViewModel()
	err = templates["stats_csv"].Execute(f, vm)
	if err != nil {
		panic(err)
	}
}

func restoreStats() {
	f, err := os.Open(fileSystemRoot + "stats.csv")
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, row := range(records) {
		page := row[0]
		count, err := strconv.Atoi(row[1])
		if(err != nil) {
			panic(err)
		}
		hitCountByPage[page] = count
	}
}

type galleryViewModel struct {
	Galleries []galleryLinkViewModel
	Images    []string
	Blurb     template.HTML
}

type indexViewModel struct {
	Galleries []galleryLinkViewModel
	About     template.HTML
}

type galleryLinkViewModel struct {
	Name         string
	PreviewImage string
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusFound)
}

func galleryHandler(w http.ResponseWriter, r *http.Request) {

	gallery, err := url.QueryUnescape(r.RequestURI[9:])

	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/", 302)
		return
	}

	incrementHitCount(gallery)

	g := galleryViewModel{
		Galleries: getGalleries(),
		Images:    getImages(gallery),
		Blurb:     getGalleryBlurb(gallery),
	}

	renderTemplate("gallery", g, w)
}

func getGalleryBlurb(gallery string) template.HTML {
	filename := fmt.Sprintf(fileSystemRoot + "galleries/%v/blurb.markdown", gallery)
	return getBlurb(filename)
}

func getBlurb(filename string) template.HTML {
	markdown, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return ""
	}

	html := template.HTML(blackfriday.MarkdownCommon(markdown))
	return html
}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	incrementHitCount("index")

	vm := indexViewModel{
		Galleries: getGalleries(),
		About:     getBlurb(fileSystemRoot + "about.markdown"),
	}

	renderTemplate("index", vm, w)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	vm := getStatsPageViewModel()
	renderTemplate("stats", vm, w)
}

func getGalleries() []galleryLinkViewModel {
	result := make([]galleryLinkViewModel, 0)
	infos, err := ioutil.ReadDir(fileSystemRoot + "galleries")
	if err != nil {
		log.Println(err)
		return result
	}

	for _, info := range infos {
		if info.IsDir() {

			galleryLinkViewModel := galleryLinkViewModel{
				Name:         info.Name(),
				PreviewImage: "/galleries/" + info.Name() + "/preview.jpg",
			}

			result = append(result, galleryLinkViewModel)
		}
	}

	return result
}

func getImages(gallery string) []string {

	result := make([]string, 0)
	dir := path.Join(fileSystemRoot + "galleries", gallery)
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Println(err)
		return result
	}

	for _, info := range infos {
		if path.Base(info.Name()) != "preview.jpg" && path.Ext(info.Name()) == ".jpg" || path.Ext(info.Name()) == ".JPG" {
			result = append(result, fmt.Sprintf("/galleries/%v/%v", gallery, info.Name()))
		}
	}

	return result
}

func renderTemplate(tmpl string, model interface{}, w http.ResponseWriter) {

	err := templates[tmpl].Execute(w, model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type pageHitCountViewModel struct {
	Page string
	HitCount int
}

type statsPageViewModel struct {
	PageHitCounts []pageHitCountViewModel
}

func incrementHitCount(page string) {
	hitCountModifyLock.Lock()
	defer saveStats()
	defer hitCountModifyLock.Unlock()

	hitCount := hitCountByPage[page]
	hitCountByPage[page] = hitCount + 1

	totalHitCount := hitCountByPage["total"]
	hitCountByPage["total"] = totalHitCount + 1
}

func getStatsPageViewModel() statsPageViewModel {
	hitCountModifyLock.Lock()
	defer hitCountModifyLock.Unlock()

	result := make([]pageHitCountViewModel, 0)
	for page, hitCount := range hitCountByPage {
		pageHitCount := pageHitCountViewModel {
			Page: page,
			HitCount: hitCount,
		}
		result = append(result, pageHitCount)
	}

	sort.Sort(ByHits(result))

	return statsPageViewModel {
		PageHitCounts: result,
	}
}

type ByHits []pageHitCountViewModel
func (a ByHits) Len() int           { return len(a) }
func (a ByHits) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHits) Less(i, j int) bool { return a[i].HitCount > a[j].HitCount }
