package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"

	"cloud.google.com/go/datastore"
)

const cookieKind = "SessionCookie"

const tokenSize = 256

type sessionCookie struct {
	User  string
	Email string
	Token string
}

func lookupCookie(ctx context.Context, c *datastore.Client, token string) (key *datastore.Key, cookie *sessionCookie, err error) {
	query := datastore.NewQuery(cookieKind).Filter("Token=", token).Limit(1)

	it := c.Run(ctx, query)

	cookie = new(sessionCookie)

	key, err = it.Next(cookie)

	return
}

func getUser(ctx context.Context, c *datastore.Client, token string) (user string, email string, err error) {
	_, cookie, err := lookupCookie(ctx, c, token)

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

	_, err = c.Put(ctx, datastore.IncompleteKey(cookieKind, nil), &cookie)
	return
}

func removeToken(ctx context.Context, c *datastore.Client, token string) (err error) {
	key, _, err := lookupCookie(ctx, c, token)

	if err != nil {
		return
	}

	err = c.Delete(ctx, key)

	return
}
