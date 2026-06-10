package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
	"testing"
)

func TestGalleryTemplateVirtualSlideshow(t *testing.T) {
	tmpl, err := template.ParseFiles("layout.html", "gallery.html")
	if err != nil {
		t.Fatal(err)
	}

	urls := []string{"/galleries/demo/a.jpg", "/galleries/demo/b.jpg", "/galleries/demo/c.jpg"}
	imagesJSON, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "layout.html", galleryViewModel{
		ImagesJSON: template.JS(imagesJSON),
		Blurb:      template.HTML("<p>test</p>"),
	})
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, `{{range`) {
		t.Error("template still contains range over images")
	}
	if !strings.Contains(out, `id="gallery-images"`) {
		t.Error("missing gallery-images JSON script")
	}
	if !strings.Contains(out, "/galleries/demo/a.jpg") {
		t.Error("missing marshalled image URL in output")
	}
	if strings.Contains(out, `<img src="/galleries/`) {
		t.Error("server should not render per-image img tags")
	}
	if !strings.Contains(out, "gallery-slideshow.js") {
		t.Error("missing gallery-slideshow.js script")
	}
	if strings.Contains(out, "jssor") {
		t.Error("jssor references should be removed from gallery template")
	}
	if strings.Contains(out, "bootstrap") {
		t.Error("bootstrap references should be removed")
	}
	if strings.Contains(out, "jquery") {
		t.Error("jquery references should be removed")
	}
	if !strings.Contains(out, "w3.css") {
		t.Error("missing w3.css stylesheet")
	}
}

func TestCatalogTemplate(t *testing.T) {
	tmpl, err := template.ParseFiles("layout.html", "catalog.html")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "layout.html", catalogViewModel{
		RoomName: "Demo Room",
		Images: []catalogImageViewModel{
			{URL: "/galleries/demo/a.jpg", Filename: "a.jpg"},
			{URL: "/galleries/demo/b.jpg", Filename: "b.jpg"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<h1 class=\"w3-container\"><a href=\"/catalogIndex\">Catalog</a> > Demo Room</h1>") {
		t.Error("missing room name in h1")
	}
	if !strings.Contains(out, "<title>Chez Watts Gallery</title>") {
		t.Error("missing default page title")
	}
	if !strings.Contains(out, "a.jpg") || !strings.Contains(out, "b.jpg") {
		t.Error("missing filenames in output")
	}
	if !strings.Contains(out, `href="/galleries/demo/a.jpg" target="_blank"`) {
		t.Error("missing target=_blank link to image")
	}
	if strings.Contains(out, "gallery-slideshow") {
		t.Error("catalog should not include slideshow")
	}
	if strings.Contains(out, `id="gallery-images"`) {
		t.Error("catalog should not include gallery-images JSON script")
	}
}
