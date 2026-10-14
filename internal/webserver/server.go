
package webserver

import (
	"os"

	"github.com/ucy-coast/websearch/internal/invertedindex"
)

type WebServer struct {
	htmlPath string
	topK     int
	index    *invertedindex.InvertedIndex
	cache	 []byte
}

func NewWebServer(htmlPath string, topK int, index *invertedindex.InvertedIndex) (*WebServer, error) {
	cachedHtml, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, err
	}

	s := &WebServer{
		htmlPath: htmlPath,
		topK:     topK,
		index:    index,
		cache:	  cachedHtml,
	}

	return s, nil
}
