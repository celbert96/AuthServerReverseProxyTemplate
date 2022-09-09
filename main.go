package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	Username    string `json:"username"`
	CreatedDate string `json:"createdDate"`
	Roles       []int  `json:"roles"`
}

// NewProxy takes target host and creates a reverse proxy
func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ModifyResponse = modifyResponse()
	return proxy, nil
}

// ProxyRequestHandler handles the http request using proxy
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

func modifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		var loginResponse LoginResponse
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Println(err.Error())
			return err
		}

		err = json.Unmarshal(body, &loginResponse)

		if err == nil {
			cookie := http.Cookie{
				Name:     "authtoken",
				Value:    loginResponse.Token,
				Expires:  time.Now().Add(365 * 24 * time.Hour),
				HttpOnly: true,
			}

			loginResponse.Token = ""
			userJson, err := json.Marshal(&loginResponse.User)

			if err != nil {
				log.Println(err.Error())
				return nil
			}

			resp.Header.Add("Set-Cookie", cookie.String())
			resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(userJson)))
			return nil
		}

		resp.Body = ioutil.NopCloser(bytes.NewBufferString(string(body)))
		return nil
	}
}

func main() {
	// initialize a reverse proxy and pass the actual backend server url here
	proxy, err := NewProxy("https://localhost:7023")
	if err != nil {
		panic(err)
	}

	// handle all requests to your server using the proxy
	http.HandleFunc("/", ProxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
