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
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"syscall"
	"time"

	"github.com/gorilla/feeds"
)

type Generator struct {
	WWWDataPath  string
	DirPath      string
	PostsPerPage int
	AnalyticsId  string

	tpl *template.Template
}

type GeneratorData struct {
	AnalyticsId string
}

type GeneratorFeedData struct {
	Feed      *Feed
	FeedTitle template.HTML
}

type GeneratorFeedsData struct {
	AnalyticsId string

	Feeds []*GeneratorFeedData
}

type GeneratorPostData struct {
	Feed *Feed

	Post          *Post
	PostAuthor    template.HTML
	PostAgeString string
	PostContent   template.HTML
}

type GeneratorPostsData struct {
	AnalyticsId string

	Feeds map[int64]*Feed

	Posts []GeneratorPostData

	Page         int
	PreviousPage int
	NextPage     int
	LastPage     int

	LastUpdate time.Time
}

func NewGenerator() *Generator {
	return &Generator{
		DirPath:      "/tmp/planetgolang",
		PostsPerPage: 10,
	}
}

func (g *Generator) Generate(tx *sql.Tx) error {
	// Prepare the output directory
	if err := os.RemoveAll(g.DirPath); err != nil {
		return fmt.Errorf("cannot delete %s: %v", g.DirPath, err)
	}

	if err := os.MkdirAll(g.DirPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %v",
			g.DirPath, err)
	}

	subDirNames := []string{"js", "css", "img", "fonts"}
	for _, name := range subDirNames {
		subDirPath := path.Join(g.DirPath, name)

		if err := os.MkdirAll(subDirPath, 0755); err != nil {
			return fmt.Errorf("cannot create directory %s: %v",
				subDirPath, err)
		}
	}

	// Copy static files
	for _, subDirName := range subDirNames {
		srcDirPath := path.Join(g.WWWDataPath, subDirName)
		files, err := ioutil.ReadDir(srcDirPath)
		if err != nil {
			if terr, ok := err.(*os.PathError); ok {
				if terr.Err == syscall.ENOENT {
					continue
				}
			}

			return fmt.Errorf("cannot list directory %s",
				srcDirPath)
		}

		for _, file := range files {
			ipath := path.Join(srcDirPath, file.Name())
			opath := path.Join(g.DirPath, subDirName, file.Name())

			if err := CopyFile(ipath, opath); err != nil {
				return err
			}
		}
	}

	// Load feeds
	var fl FeedList
	if err := fl.LoadEnabled(tx); err != nil {
		return err
	}

	feeds := make(map[int64]*Feed)
	for _, f := range fl {
		feeds[f.Id] = f
	}

	// Count posts
	nbPosts, err := CountPosts(tx)
	if err != nil {
		return err
	}

	// Load templates
	g.tpl = template.Must(template.ParseFiles(
		"tpl/main.tmpl",
		"tpl/feeds.tmpl",
		"tpl/about.tmpl",
		"tpl/posts.tmpl"))

	// Generate the feed page
	sort.Sort(fl)

	feedsData := &GeneratorFeedsData{
		AnalyticsId: g.AnalyticsId,
		Feeds:       make([]*GeneratorFeedData, len(fl)),
	}

	for i, f := range fl {
		feedsData.Feeds[i] = &GeneratorFeedData{
			Feed:      f,
			FeedTitle: template.HTML(f.Title),
		}
	}

	if err := g.GeneratePage("feeds.html", "feeds", feedsData); err != nil {
		return err
	}

	// Generate the about page
	aboutData := &GeneratorData{
		AnalyticsId: g.AnalyticsId,
	}

	if err := g.GeneratePage("about.html", "about", aboutData); err != nil {
		return err
	}

	// Generate post pages
	offset := 0
	page := 1
	for {
		var posts PostList
		err := posts.LoadRange(tx, g.PostsPerPage, offset)
		if err != nil {
			return err
		}
		if len(posts) == 0 {
			break
		}

		postsData := make([]GeneratorPostData, len(posts))
		for i, post := range posts {
			feed := feeds[post.FeedId]

			var author string
			if post.Author != "" {
				author = post.Author
			} else if feed.Author != "" {
				author = feed.Author
			} else {
				author = feed.Title
			}

			postsData[i] = GeneratorPostData{
				Feed: feed,

				Post:          post,
				PostAuthor:    template.HTML(author),
				PostAgeString: post.AgeString(),
				PostContent:   template.HTML(post.Content),
			}
		}

		data := GeneratorPostsData{
			AnalyticsId: g.AnalyticsId,

			Posts: postsData,

			Page:         page,
			PreviousPage: page - 1,
			NextPage:     page + 1,
			LastPage:     nbPosts/g.PostsPerPage + 1,

			LastUpdate: time.Now(),
		}

		pageName := fmt.Sprintf("page-%05d.html", page)
		if err := g.GeneratePage(pageName, "posts", data); err != nil {
			return err
		}

		offset += len(posts)
		page++
	}

	// Link index.html to the first post page
	indexPage := path.Join(g.DirPath, "index.html")
	firstPostsPage := "page-00001.html"

	if err := os.Symlink(firstPostsPage, indexPage); err != nil {
		return fmt.Errorf("cannot symlink %s to %s: %v",
			indexPage, firstPostsPage, err)
	}

	// Generate the RSS feed
	if err := g.GenerateFeed(tx, "rss.xml"); err != nil {
		return fmt.Errorf("cannot generate rss feed: %v", err)
	}

	return nil
}

func (g *Generator) GeneratePage(filePath string, tplName string, data interface{}) error {
	filePath = path.Join(g.DirPath, filePath)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %v", filePath, err)
	}
	defer file.Close()

	if err := g.tpl.ExecuteTemplate(file, tplName, data); err != nil {
		return fmt.Errorf("cannot execute template %s: %v",
			tplName, err)
	}

	return nil
}

func (g *Generator) GenerateFeed(tx *sql.Tx, filePath string) error {
	// Load last posts
	var posts PostList
	err := posts.LoadRange(tx, 10, 0)
	if err != nil {
		return err
	}

	// Generate feed items
	items := make([]*feeds.Item, len(posts))
	for i, post := range posts {
		items[i] = &feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: post.URL},
			Id:          post.URL,
			Author:      &feeds.Author{Name: post.Author},
			Created:     post.Date,
			Description: post.Content,
		}
	}

	// Generate feed
	now := time.Now()

	feed := &feeds.Feed{
		Title:       "Planet Golang",
		Link:        &feeds.Link{Href: "http://planetgolang.com"},
		Description: "An aggregator of various Go-related blogs.",
		Created:     now,
		Items:       items,
	}

	rss, err := feed.ToRss()
	if err != nil {
		return fmt.Errorf("cannot generate rss feed: %v", err)
	}

	// Write it
	filePath = path.Join(g.DirPath, filePath)
	if err := ioutil.WriteFile(filePath, []byte(rss), 0644); err != nil {
		return fmt.Errorf("cannot write %s: %v", filePath, err)
	}

	return nil
}
