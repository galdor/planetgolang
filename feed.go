// Copyright (c) 2016 Nicolas Martyanoff <khaelin@gmail.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"database/sql"
	"fmt"

	"github.com/mmcdole/gofeed"
)

type Feed struct {
	Id         int64
	URL        string
	Title      string
	Author     string // optional
	WebsiteURL string
	Enabled    bool

	feed *gofeed.Feed
}

type FeedList []*Feed

func (f *Feed) Insert(tx *sql.Tx) error {
	res, err := tx.Exec(
		`INSERT INTO feeds (url, title, author, website_url, enabled)
		   VALUES (?, ?, ?, ?, ?)`,
		f.URL, f.Title, f.Author, f.WebsiteURL, f.Enabled)
	if err != nil {
		return fmt.Errorf("cannot insert feed: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("cannot retrieve feed id: %v", err)
	}

	f.Id = id
	return nil
}

func (f *Feed) Update(tx *sql.Tx) error {
	_, err := tx.Exec(
		`UPDATE feeds SET
		     url = ?,
		     title = ?,
		     author = ?,
		     website_url = ?,
		     enabled = ?
		   WHERE id = ?`,
		f.URL, f.Title, f.Author, f.WebsiteURL, f.Enabled,
		f.Id)
	if err != nil {
		return fmt.Errorf("cannot update feed: %v", err)
	}

	return nil
}

func (f *Feed) Download() error {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(f.URL)
	if err != nil {
		return fmt.Errorf("cannot download %s: %v", f.URL, err)
	}

	f.feed = feed
	return nil
}

func (f *Feed) ExtractMetadata() {
	if f.feed.Title != "" {
		f.Title = f.feed.Title
	}

	if f.feed.Author != nil && f.feed.Author.Name != "" {
		f.Author = f.feed.Author.Name
	}

	if f.feed.Link != "" {
		f.WebsiteURL = f.feed.Link
	}
}

func (f *Feed) ExtractPosts() PostList {
	var ps PostList

	for _, item := range f.feed.Items {
		p := &Post{FeedId: f.Id, Enabled: true}

		p.ReadFromGofeedItem(item)
		if p.URL == "" || p.Title == "" || p.Date.IsZero() {
			continue
		}

		ps = append(ps, p)
	}

	return ps
}

func (f *Feed) ReadFromRow(row *sql.Rows) error {
	return row.Scan(&f.Id, &f.URL, &f.Title, &f.Author, &f.WebsiteURL,
		&f.Enabled)
}

func (fl FeedList) Len() int           { return len(fl) }
func (fl FeedList) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl FeedList) Less(i, j int) bool { return fl[i].Title < fl[j].Title }

func (fl *FeedList) LoadEnabled(tx *sql.Tx) error {
	rows, err := tx.Query(
		`SELECT id, url, title, author, website_url, enabled
		   FROM feeds
		   WHERE enabled = 1`)
	if err != nil {
		return fmt.Errorf("cannot load feeds: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		f := &Feed{}
		if err := f.ReadFromRow(rows); err != nil {
			return fmt.Errorf("invalid feed: %v", err)
		}

		*fl = append(*fl, f)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("cannot load feeds: %v", err)
	}

	return nil
}
