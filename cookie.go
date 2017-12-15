package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

const cookieKind = "SessionCookie"

const tokenSize = 256

const tokenTimeout = time.Hour

var errorTokenErpired = errors.New("Token Expired")

type sessionCookie struct {
	User   string
	Email  string
	Token  string
	Expiry time.Time
}

const cookieCleanPath = "/cleanup"

func getUser(ctx context.Context, c *datastore.Client, token string) (user string, email string, err error) {
	query := datastore.NewQuery(cookieKind).Filter("Token =", token).Limit(1)

	it := c.Run(ctx, query)

	var cookie sessionCookie

	_, err = it.Next(&cookie)

	if err != nil {
		log.Println(err)
		return
	}

	if time.Now().After(cookie.Expiry) {
		err = errorTokenErpired
		return
	}

	user = cookie.User
	email = cookie.Email
	return
}

func genToken(ctx context.Context, c *datastore.Client, user string, email string) (token string, err error) {
	var tokenBytes [tokenSize]byte

	_, err = io.ReadFull(rand.Reader, tokenBytes[:])

	if err != nil {
		return
	}

	token = base64.StdEncoding.EncodeToString(tokenBytes[:])

	cookie := sessionCookie{
		User:   user,
		Email:  email,
		Token:  token,
		Expiry: time.Now().Add(tokenTimeout),
	}

	key, err := c.Put(ctx, datastore.IncompleteKey(cookieKind, nil), &cookie)

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		if err := c.Delete(context.Background(), key); err != nil {
			log.Println("Error removing token:", err)
			return
		}
	}()

	return
}

func cleanupExpiredCookies(ctx context.Context, c *datastore.Client) error {
	query := datastore.NewQuery(cookieKind)

	it := c.Run(ctx, query)

	var cookie sessionCookie

	keys := []*datastore.Key{}

	for {
		key, err := it.Next(&cookie)

		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		if time.Now().After(cookie.Expiry) {
			keys = append(keys, key)
		}
	}

	return c.DeleteMulti(ctx, keys)
}

type cleanCookiesHandler struct {
	c *datastore.Client
}

func (h *cleanCookiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := cleanupExpiredCookies(r.Context(), h.c); err != nil {
		log.Println("Couldn't clean expired cookies:", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	fmt.Fprintln(w, "Cleaned Expired Cookies")
}

//CleanCookiesHandler handles expired cookies cleanup
func CleanCookiesHandler(c *datastore.Client) (string, http.Handler) {
	return cookieCleanPath, &cleanCookiesHandler{
		c: c,
	}
}
