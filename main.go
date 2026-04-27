// Command distraction.today serves the daily quote site.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/feeds"
	"github.com/icco/distraction.today/static"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unrolled/render"
	"github.com/unrolled/secure"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.uber.org/zap"
)

//go:embed templates
var embeddedTemplates embed.FS

const service = "distraction.today"

var (
	log = logging.Must(logging.NewLogger(service))
	re  = render.New(render.Options{
		Layout:                    "layout",
		Charset:                   "UTF-8",
		DisableHTTPErrorRendering: false,
		Extensions:                []string{".tmpl", ".html"},
		RequirePartials:           false,
		Funcs:                     []template.FuncMap{},
		FileSystem:                &render.EmbedFileSystem{FS: embeddedTemplates},
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

	registry := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		log.Errorw("otel prometheus exporter", zap.Error(err))
		os.Exit(1)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(mp)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mp.Shutdown(shutdownCtx); err != nil {
			log.Warnw("meter provider shutdown", zap.Error(err))
		}
	}()

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
	r.Use(logging.Middleware(log.Desugar()))
	r.Use(routeTag)
	r.Use(etag.Handler(false))
	r.Use(secureMiddleware.Handler)

	crs := cors.New(cors.Options{
		AllowCredentials:   false,
		OptionsPassthrough: false,
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:     []string{"Link"},
		MaxAge:             300,
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
		if _, err := w.Write([]byte("hi.")); err != nil {
			logging.FromContext(r.Context()).Errorw("write healthz", zap.Error(err))
		}
	})

	r.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		date := time.Now().Format("2006-01-02")
		if _, err := static.GetTodaysQuote(time.Now()); err != nil {
			l.Debugw("no quote for today, falling back to latest", zap.Error(err))
			if q, err := static.GetLatestQuote(); err == nil {
				date = q.Date
			} else {
				l.Errorw("failed to find latest quote", zap.Error(err))
			}
		}
		http.Redirect(w, r, fmt.Sprintf("/%s", date), http.StatusTemporaryRedirect)
	})

	r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		data := TemplateData{
			Year:  time.Now().Year(),
			Title: "distraction.today | about",
		}

		if err := re.HTML(w, http.StatusOK, "about", data); err != nil {
			l.Errorw("render about page", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	r.Get("/{year}-{month}-{day}", func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		year, month, day := chi.URLParam(r, "year"), chi.URLParam(r, "month"), chi.URLParam(r, "day")
		date := fmt.Sprintf("%s-%s-%s", year, month, day)
		datetime, err := time.Parse("2006-01-02", date)
		if err != nil {
			l.Infow("invalid date", "date", date, zap.Error(err))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		q, err := static.GetTodaysQuote(datetime)
		if err != nil {
			l.Debugw("no quote for date, rendering 404", "date", date, zap.Error(err))
			data := TemplateData{
				Year:  time.Now().Year(),
				Title: "distraction.today | 404",
			}
			if renderErr := re.HTML(w, http.StatusNotFound, "404", data); renderErr != nil {
				l.Errorw("render 404 page", zap.Error(renderErr))
			}
			return
		}

		data := TemplateData{
			Quote:          q,
			ContributorURL: static.GetContribURL(q.Contributor),
			Year:           time.Now().Year(),
			Title:          fmt.Sprintf("distraction.today | %s", date),
		}

		if err := re.HTML(w, http.StatusOK, "index", data); err != nil {
			l.Errorw("render index page", "date", date, zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	r.Get("/feed.rss", func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		feed, err := generateFeed()
		if err != nil {
			l.Errorw("generate rss feed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := feed.ToRss()
		if err != nil {
			l.Errorw("render rss feed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/rss+xml")
		if err := re.Text(w, http.StatusOK, data); err != nil {
			l.Errorw("write rss feed", zap.Error(err))
		}
	})

	r.Get("/feed.atom", func(w http.ResponseWriter, r *http.Request) {
		l := logging.FromContext(r.Context())
		feed, err := generateFeed()
		if err != nil {
			l.Errorw("generate atom feed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := feed.ToAtom()
		if err != nil {
			l.Errorw("render atom feed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/atom+xml")
		if err := re.Text(w, http.StatusOK, data); err != nil {
			l.Errorw("write atom feed", zap.Error(err))
		}
	})

	handler := otelhttp.NewHandler(r, service,
		otelhttp.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/metrics"
		}),
	)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorw("http server", zap.Error(err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorw("http shutdown", zap.Error(err))
	}
}

// routeTag stamps the chi route pattern onto otelhttp metric labels.
func routeTag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		labeler, ok := otelhttp.LabelerFromContext(r.Context())
		if !ok {
			return
		}
		if pattern := chi.RouteContext(r.Context()).RoutePattern(); pattern != "" {
			labeler.Add(semconv.HTTPRoute(pattern))
		}
	})
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
