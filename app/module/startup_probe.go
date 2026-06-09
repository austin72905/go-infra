package module

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// app 正式開始 startup 之前，先確認外部依賴是不是已經可用。
// 原本的readinessProbe 命名不佳
const startupProbeTimeout = 15 * time.Second

type StartupProbe struct {
	hostURIs []string
	urls     []string
}

func NewStartupProbe() *StartupProbe {
	return &StartupProbe{}
}

func (p *StartupProbe) AddHostURI(hostURI string) {
	p.hostURIs = append(p.hostURIs, hostURI)
}

func (p *StartupProbe) AddURL(url string) {
	p.urls = append(p.urls, url)
}

// 並發檢查
func (p *StartupProbe) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, startupProbeTimeout)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(p.hostURIs)+len(p.urls))

	for _, hostURI := range p.hostURIs {
		hostURI := hostURI
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.resolveHost(ctx, hostname(hostURI)); err != nil {
				errCh <- err
			}
		}()
	}

	for _, url := range p.urls {
		url := url
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.checkHTTP(ctx, url); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *StartupProbe) resolveHost(ctx context.Context, host string) error {
	resolver := net.Resolver{}
	_, err := resolver.LookupHost(ctx, host)
	if err != nil {
		return fmt.Errorf("startup probe failed to resolve host %q: %w", host, err)
	}
	return nil
}

func (p *StartupProbe) checkHTTP(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("startup probe invalid url %q: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("startup probe failed for url %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("startup probe got status %d for url %q", resp.StatusCode, url)
	}
	return nil
}

func hostname(hostURI string) string {
	index := strings.Index(hostURI, ":")
	if index == -1 {
		return hostURI
	}
	return hostURI[:index]
}
