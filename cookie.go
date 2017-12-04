package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"

	"cloud.google.com/go/datastore"
)

const cookieKind = "SessionCookie"

const tokenSize = 256

type sessionCookie struct {
	User  string
	Email string
	Token string
}

func getUser(ctx context.Context, c *datastore.Client, token string) (user string, email string, err error) {
	query := datastore.NewQuery(cookieKind).Filter("Token=", token).Limit(1)

	it := c.Run(ctx, query)

	var cookie sessionCookie

	_, err = it.Next(&cookie)

	if err != nil {
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
		User:  user,
		Email: email,
		Token: token,
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
