package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/russross/blackfriday"
)

const portHttp = 8200
const fileSystemRoot = "/home/ubuntu/data/chezwatts.gallery/"
const statsLogFilename = "stats_log.csv"
const statsFilename = "stats.csv"
const statsTemplateFilename = "stats.csv.tmpl"

var templates = make(map[string]*template.Template)
var hitCountByPage = make(map[string]int)
var hitCountModifyLock = &sync.Mutex{}

func main() {

	defer saveStats()

	restoreStats()

	httpMux := http.NewServeMux()

	httpMux.HandleFunc("/favicon.ico", faviconHandler)
	httpMux.HandleFunc("/", indexHandler)
	httpMux.HandleFunc("/gallery/", galleryHandler)
	httpMux.HandleFunc("/stats", statsHandler)
	httpMux.Handle("/galleries/", http.StripPrefix("/galleries/", http.FileServer(http.Dir(fileSystemRoot+"galleries"))))
	httpMux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(fileSystemRoot+"js"))))
	httpMux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(fileSystemRoot+"css"))))
	httpMux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir(fileSystemRoot+"img"))))
	httpMux.HandleFunc("/stats-log", statsLogHandler)

	go updateStatsLogDaily()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(portHttp), logAndDelegate(httpMux)))
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

	t, err := template.ParseFiles(fileSystemRoot + statsTemplateFilename)
	if err != nil {
		panic(err)
	}

	templates["stats_csv"] = t
}

func updateStatsLogDaily() {
	c := time.Tick(24 * time.Hour)
	for range c {
		updateStatsLog()
	}
}

func updateStatsLog() {
	log.Println("1")
	// hitCountModifyLock.Lock()
	// defer hitCountModifyLock.Unlock()

	records := make([][]string, 0)

	filename := fileSystemRoot + statsLogFilename

	log.Println("2")
	f, err := os.Open(filename)

	log.Println("3")
	if err != nil {
		log.Println("4")
		if os.IsNotExist(err) {
			log.Println("5")
			// if stats log file doesn't exist then
			// records is minimal header row and no record rows
			headerRow := []string{"Date"}
			records = append(records, headerRow)
		} else {
			log.Println("6")
			log.Fatal(err)
		}
	} else {
		log.Println("7")
		// if stats log file exists
		defer f.Close()

		// records = read stats log file
		r := csv.NewReader(f)
		records2, err := r.ReadAll()
		if err != nil {
			log.Fatal(err)
		} else {
			records = records2
		}
	}

	log.Println("8")

	// create new empty record with current date
	headerRow := records[0]
	numCols := len(headerRow)
	newRecord := make([]string, numCols)
	newRecord[0] = fmt.Sprint(time.Now().Date())

	log.Println("9")
	// for each gallery in stats
	stats := getStatsPageViewModel()

	log.Println("10")
	for _, gallery := range stats.PageHitCounts {

		columnIndex := indexOf(gallery.Page, headerRow)
		// if not exists as a column in the log
		if columnIndex < 0 {
			// append name to header record
			headerRow = append(headerRow, gallery.Page)
			records[0] = headerRow
			newRecord = append(newRecord, "0")

			// append zero to each other record
			for i := range records {
				if i == 0 {
					continue
				}

				records[i] = append(records[i], "0")
			}

			columnIndex = len(headerRow) - 1
		}

		// set correct field of the new record
		newRecord[columnIndex] = fmt.Sprint(gallery.HitCount)
	}

	log.Println("11")

	records = append(records, newRecord)

	// overwrite file
	f.Close()
	f, err = os.Create(filename)
	if err != nil {
		log.Println("12")
		log.Fatal(err)
	}
	log.Println("13")
	defer f.Close()
	writer := csv.NewWriter(f)
	err = writer.WriteAll(records)
	log.Println("14")
	if err != nil {
		log.Println("15")
		log.Fatal(err)
	}
}

func indexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}

func logAndDelegate(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL.Path, r.RemoteAddr, r.Referer(), r.UserAgent())
		handler.ServeHTTP(w, r)
	})
}

func saveStats() {
	f, err := os.Create(fileSystemRoot + statsFilename)
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
	f, err := os.Open(fileSystemRoot + statsFilename)
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(f)

	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, row := range records {
		page := row[0]
		count, err := strconv.Atoi(row[1])
		if err != nil {
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

func statsLogHandler(w http.ResponseWriter, r *http.Request) {
	hitCountModifyLock.Lock()
	defer hitCountModifyLock.Unlock()
	http.ServeFile(w, r, fileSystemRoot+statsLogFilename)
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

	exists := getGalleryExists(gallery)

	if !exists {
		log.Println("Invalid request ignored.")
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
	filename := fmt.Sprintf(fileSystemRoot+"galleries/%v/blurb.markdown", gallery)
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

func getGalleryExists(gallery string) bool {
	dir := path.Join(fileSystemRoot+"galleries", gallery)

	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false
	}

	return true
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
	dir := path.Join(fileSystemRoot+"galleries", gallery)
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
	Page     string
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
		pageHitCount := pageHitCountViewModel{
			Page:     page,
			HitCount: hitCount,
		}
		result = append(result, pageHitCount)
	}

	sort.Sort(ByHits(result))

	return statsPageViewModel{
		PageHitCounts: result,
	}
}

type ByHits []pageHitCountViewModel

func (a ByHits) Len() int           { return len(a) }
func (a ByHits) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHits) Less(i, j int) bool { return a[i].HitCount > a[j].HitCount }
