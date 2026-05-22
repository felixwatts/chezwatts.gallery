package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
	"testing"
)

func TestGalleryTemplateVirtualSlideshow(t *testing.T) {
	tmpl, err := template.ParseFiles("gallery.html")
	if err != nil {
		t.Fatal(err)
	}

	urls := []string{"/galleries/demo/a.jpg", "/galleries/demo/b.jpg", "/galleries/demo/c.jpg"}
	imagesJSON, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, galleryViewModel{
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
}
