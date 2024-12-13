package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/feeds"
	"github.com/icco/distraction.today/static"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/unrolled/render"
	"github.com/unrolled/secure"
)

const (
	service = "distraction.today"
	project = "icco-cloud"
)

var (
	log = logging.Must(logging.NewLogger(service))
	re  = render.New(render.Options{
		Layout:                    "layout",
		Charset:                   "UTF-8",
		DisableHTTPErrorRendering: false,
		Extensions:                []string{".tmpl", ".html"},
		RequirePartials:           false,
		Funcs:                     []template.FuncMap{},
	})
)

type TemplateData struct {
	Quote          *static.Quote
	ContributorURL string
	Year           int
	Title          string
}

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:        false,
		SSLProxyHeaders:    map[string]string{"X-Forwarded-Proto": "https"},
		FrameDeny:          true,
		ContentTypeNosniff: true,
		BrowserXssFilter:   true,
		ReferrerPolicy:     "no-referrer",
		FeaturePolicy:      "geolocation 'none'; midi 'none'; sync-xhr 'none'; microphone 'none'; camera 'none'; magnetometer 'none'; gyroscope 'none'; fullscreen 'none'; payment 'none'; usb 'none'",
	})

	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), project))
	r.Use(secureMiddleware.Handler)

	crs := cors.New(cors.Options{
		AllowCredentials:   true,
		OptionsPassthrough: false,
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:     []string{"Link"},
		MaxAge:             300, // Maximum value not ignored by any of major browsers
	})
	r.Use(crs.Handler)

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("report-to", `{"group":"default","max_age":10886400,"endpoints":[{"url":"https://reportd.natwelch.com/report/distraction"}]}`)
			w.Header().Set("reporting-endpoints", `default="https://reportd.natwelch.com/reporting/distraction"`)

			h.ServeHTTP(w, r)
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi."))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("/%s", time.Now().Format("2006-01-02"))
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})

	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		data := TemplateData{
			Year:  time.Now().Year(),
			Title: "distraction.today | about",
		}

		if err := re.HTML(w, http.StatusOK, "about", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	r.Get("/{year}-{month}-{day}", func(w http.ResponseWriter, r *http.Request) {
		year, month, day := chi.URLParam(r, "year"), chi.URLParam(r, "month"), chi.URLParam(r, "day")
		date := fmt.Sprintf("%s-%s-%s", year, month, day)
		datetime, err := time.Parse("2006-01-02", date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		q, err := static.GetTodaysQuote(datetime)
		if err != nil {
			data := TemplateData{
				Year:  time.Now().Year(),
				Title: "distraction.today | 404",
			}
			re.HTML(w, http.StatusNotFound, "404", data)
			return
		}

		data := TemplateData{
			Quote:          q,
			ContributorURL: static.GetContribURL(q.Contributor),
			Year:           time.Now().Year(),
			Title:          fmt.Sprintf("distraction.today | %s", date),
		}

		if err := re.HTML(w, http.StatusOK, "index", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	r.Get("/feed.rss", func(w http.ResponseWriter, r *http.Request) {
		feed, err := generateFeed()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := feed.ToRss()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/rss+xml")
		re.Text(w, http.StatusOK, data)
	})

	r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
		feed, err := generateFeed()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := feed.ToAtom()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		re.Text(w, http.StatusOK, data)
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Errorw("Failed to start server", "error", err)
	}
}

func generateFeed() (*feeds.Feed, error) {
	feed := &feeds.Feed{
		Title:       "distraction.today",
		Link:        &feeds.Link{Href: "https://distraction.today"},
		Description: "A daily quote to distract you.",
		Author:      &feeds.Author{Name: "Nat Welch", Email: "nat@natwelch.com"},
	}

	quotes, err := static.GetQuotes()
	if err != nil {
		return nil, err
	}

	for _, quote := range quotes {
		when, err := time.Parse("2006-01-02", quote.Date)
		if err != nil {
			return nil, err
		}

		text := fmt.Sprintf("%q \n - %s", quote.Quote, quote.Author)

		feed.Items = append(feed.Items, &feeds.Item{
			Title:   quote.Date,
			Content: text,
			Link:    &feeds.Link{Href: fmt.Sprintf("https://distraction.today/%s", quote.Date)},
			Created: when,
		})
	}

	return feed, nil
}
