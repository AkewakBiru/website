package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
)

var (
	logger   *zap.Logger
	Db       *sql.DB
	Oauthcfg *oauth2.Config
	Store    *sessions.CookieStore
	LogFile  *os.File
)

type BlogData struct {
	Articles []Article
	Size     int
	PrevPage int
	NextPage int
}

type OauthData struct {
	AccessToken string
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

func init() {
	var envFile string
	var environ bool = false
	if env := os.Getenv("APP_ENV"); env != "PROD" {
		envFile = ".env.dev"
	} else {
		envFile = ".env.prod"
		environ = true
	}

	if err := godotenv.Load(envFile); err != nil {
		log.Fatal("Error loading env file", err)
	}
	if environ {
		handle, err := os.OpenFile("/var/log/app/access.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			log.Fatal("Error opening/ creating access file", err)
		}
		LogFile = handle
		config := zap.NewProductionEncoderConfig()
		// omit stacktrace except for FATAL level where stacktrace can't be removed
		config.StacktraceKey = zapcore.OmitKey
		outCore := zapcore.NewCore(zapcore.NewJSONEncoder(config), LogFile, zap.InfoLevel)
		logger = zap.New(outCore)
	} else {
		config := zap.NewDevelopmentConfig()
		logger = zap.Must(config.Build())
	}
	defer logger.Sync()
}

func init() {
	connStr := fmt.Sprintf("oracle://%s:%s@%s:%s/%s?TRACE FILE=%s&SSL=enable&SSL Verify=false&WALLET=%s/%s",
		os.Getenv("USERNAME"), os.Getenv("PASSWORD"), os.Getenv("HOST"), os.Getenv("PORT"),
		os.Getenv("SERVICE"), os.Getenv("TRACE_PATH"), os.Getenv("PWD"), os.Getenv("WALLET_PATH"))
	db, err := sql.Open("oracle", connStr)
	DieOnError("error in sql.Open:", err)
	err = db.Ping()
	DieOnError("error in db.Ping:", err)

	Db = db
}

func init() {
	config := oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
		RedirectURL: "https://akewak.biru.etco.cloud/oauth2/callback",
		Scopes:      []string{os.Getenv("OAUTH_SCOPE")},
	}

	Oauthcfg = &config
	Store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
}

func DieOnError(msg string, err error) {
	if err != nil {
		LogFile.Close()
		logger.Fatal(msg, zap.Error(err))
	}
}

func loggingMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("HTTP request received", zap.String("Method", r.Method), zap.String("Path", r.RequestURI),
			zap.String("Remote-Addr", r.RemoteAddr), zap.String("User-Agent", r.UserAgent()))
		handler(w, r)
	}
}

func chain(f http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for _, m := range middlewares {
		f = m(f)
	}
	return f
}

func createArticle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(r.FormValue("title")) == 0 || len(r.FormValue("article")) == 0 || len(r.FormValue("slug")) == 0 {
		logger.Error("create article failed", zap.String("reason", "Invalid form value/s"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := InsertRecord(Db, Article{Title: r.FormValue("title"), Content: r.FormValue("article"),
		Slug: r.FormValue("slug"), Date_posted: time.Now()}); err != nil {
		logger.Error("create article failed", zap.Error(err))
	}
	http.Redirect(w, r, "/blog", http.StatusTemporaryRedirect)
}

// healthcheck endpoint
func test(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	defer Db.Close()
	defer LogFile.Close()

	fs := http.FileServer(http.Dir("./static"))
	r := mux.NewRouter()
	// serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs)).Methods("GET")
	r.HandleFunc("/test", chain(test, loggingMiddleware))
	r.HandleFunc("/blog", chain(blogHandler, loggingMiddleware)).Methods("GET", "POST")
	r.HandleFunc("/articleEditor", chain(articleEditorHandler, authMiddleware, loggingMiddleware))
	r.HandleFunc("/create", chain(createArticle, authMiddleware, loggingMiddleware)).Methods("POST")
	r.HandleFunc("/resume", chain(resumeHandler, loggingMiddleware))
	r.HandleFunc("/blog/{id:[0-9]+}", chain(articleHandler, loggingMiddleware))
	r.HandleFunc("/oauth2/callback", callbackHandler)

	logger.Info("Server started listening on :", zap.String("port", os.Getenv("SERVER_PORT")))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("SERVER_PORT")), r); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
