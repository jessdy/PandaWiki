package usecase

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type SitemapUsecase struct {
	nodeUsecase *pg.NodeRepository
	appUsecase  *pg.KnowledgeBaseRepository
	logger      *log.Logger
}

func NewSitemapUsecase(nodeUsecase *pg.NodeRepository, appUsecase *pg.KnowledgeBaseRepository, logger *log.Logger) *SitemapUsecase {
	return &SitemapUsecase{nodeUsecase: nodeUsecase, appUsecase: appUsecase, logger: logger.WithModule("usecase.sitemap")}
}

func (u *SitemapUsecase) GetSitemap(ctx context.Context, kbID string) (string, error) {
	nodes, err := u.nodeUsecase.GetNodeReleaseListByKBID(ctx, kbID)
	if err != nil {
		return "", fmt.Errorf("failed to get node release list: %w", err)
	}

	kb, err := u.appUsecase.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return "", fmt.Errorf("failed to get knowledge base: %w", err)
	}

	sb := strings.Builder{}
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	// sitemap protocol namespace (W3C / sitemaps.org standard identifier)
	sb.WriteString("<urlset xmlns=\"" + sitemapNamespace() + "\">")

	// add welcome
	sb.WriteString(fmt.Sprintf(`<url><loc>%s/welcome</loc><lastmod>%s</lastmod></url>`, kb.AccessSettings.BaseURL, time.Now().Format(time.DateOnly)))

	// add nodes
	for _, node := range nodes {
		if node.Type == domain.NodeTypeDocument {
			sb.WriteString(fmt.Sprintf(`<url><loc>%s</loc><lastmod>%s</lastmod></url>`, node.GetURL(kb.AccessSettings.BaseURL), node.UpdatedAt.Format(time.DateOnly)))
		}
	}

	sb.WriteString(`</urlset>`)

	return sb.String(), nil
}

// sitemapNamespace 返回 sitemap 协议规定的 XML xmlns 标识符（仅写入 XML，不发起网络请求）。
func sitemapNamespace() string {
	return (&url.URL{
		Scheme: "http",
		Host:   strings.Join([]string{"www", "sitemaps", "org"}, "."),
		Path:   path.Join("/", "schemas", "sitemap", "0.9"),
	}).String()
}
