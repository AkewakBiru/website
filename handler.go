package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type User struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

type OauthStuff struct {
	UserInfo    *User
	AccessToken string
	OAuthConfig *oauth2.Config
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	offset := 5
	templ := template.Must(template.ParseFiles(filepath.Join("./templates", "blog.html")))
	pageNum, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	rec, err := GetNumRecords(Db)
	if err != nil {
		logger.Warn("Read error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if pageNum-1 > rec/offset {
		pageNum = 1
	}
	logger.Debug("Number of records returned", zap.Int("NumRecords", rec))
	data := BlogData{
		Size:     rec,
		Articles: nil,
		PrevPage: 0,
		NextPage: 0,
	}
	if pageNum*offset < rec {
		data.NextPage = pageNum + 1
	}
	if pageNum > 1 {
		data.PrevPage = pageNum - 1
	}
	res, err := GetNRecords(Db, pageNum-1, offset)
	if err != nil {
		logger.Warn("Read error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data.Articles = res
	templ.Execute(w, data)
}

func articleEditorHandler(w http.ResponseWriter, r *http.Request) {
	templ := template.Must(template.ParseFiles(filepath.Join("templates", "articleEditor.html")))
	templ.Execute(w, r)
}

func resumeHandler(w http.ResponseWriter, r *http.Request) {
	templ := template.Must(template.ParseFiles(filepath.Join("templates", "resume.html")))
	templ.Execute(w, r)
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Warn("Invalid request", zap.String("reason", "invalid article id"), zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	article, err := GetRecord(Db, id)
	if err != nil {
		logger.Warn("Invalid request", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	templ := template.Must(template.ParseFiles(filepath.Join("templates", "article.html")))
	templ.Execute(w, article)
}

func authMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		OauthConfig := OauthStuff{OAuthConfig: Oauthcfg}
		templ := template.Must(template.New("oauth.html").Funcs(template.FuncMap{"join": strings.Join}).ParseFiles("templates/oauth.html"))
		session, err := Store.Get(r, "auth-cookie")
		if err != nil {
			logger.Warn("Error retrieving cookie store", zap.Error(err))
			if err := templ.Execute(w, OauthConfig); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logger.Warn("Error executing template", zap.Error(err))
			}
			return
		}
		if token, ok := session.Values["Access-Token"].(string); ok {
			OauthConfig.AccessToken = token
		} else {
			logger.Warn("Access token doesn't exist")
			if err := templ.Execute(w, OauthConfig); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logger.Warn("Error executing template", zap.Error(err))
			}
			return
		}

		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
		if err != nil {
			logger.Warn("Failed creating request", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OauthConfig.AccessToken))
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Warn("Error getting a response", zap.Error(err))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		defer resp.Body.Close()

		var user User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			logger.Warn("JSON decoding failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if user.Login != "AkewakBiru" {
			logger.Warn("User not authorized")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := Oauthcfg.Exchange(r.Context(), code)
	if err != nil {
		logger.Warn("Error exchanging code for token", zap.Error(err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	logger.Info("Oauth token exchange successful", zap.String("token type", token.TokenType))
	// save the access token in a cookie
	session, err := Store.Get(r, "auth-cookie")
	if err != nil {
		logger.Warn("Error retrieving cookie store", zap.Error(err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	session.Values["Access-Token"] = token.AccessToken
	session.Save(r, w)
	w.Header().Set("Access-Token", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	http.Redirect(w, r, "/articleEditor", http.StatusTemporaryRedirect)
}
