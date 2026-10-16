package tufclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata"
)

const (
	RootMaxLength      int64 = 512 * 1024
	TimestampMaxLength int64 = 64 * 1024
	SnapshotMaxLength  int64 = 1024 * 1024
	TargetsMaxLength   int64 = 4 * 1024 * 1024
	TargetMaxLength    int64 = 128 * 1024 * 1024
	MaxRootRotations   int64 = 32
	MaxRedirects             = 3
	MaxRetries               = 2
)

type Transport struct {
	client          *http.Client
	allowedHosts    map[string]struct{}
	userAgent       string
	metadataTimeout time.Duration
	targetTimeout   time.Duration
}

type TransportOption func(*transportOptions)

type transportOptions struct {
	roundTripper http.RoundTripper
}

func WithRoundTripper(roundTripper http.RoundTripper) TransportOption {
	return func(options *transportOptions) { options.roundTripper = roundTripper }
}

func NewTransport(metadataURL, targetsURL, userAgent string, options ...TransportOption) (*Transport, error) {
	allowed := make(map[string]struct{}, 2)
	for _, raw := range []string{metadataURL, targetsURL} {
		u, err := validateRemoteURL(raw)
		if err != nil {
			return nil, err
		}
		allowed[u.Host] = struct{}{}
	}
	settings := transportOptions{}
	for _, option := range options {
		option(&settings)
	}
	roundTripper := settings.roundTripper
	if roundTripper == nil {
		roundTripper = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			ForceAttemptHTTP2:     true,
		}
	}
	transport := &Transport{
		allowedHosts:    allowed,
		userAgent:       userAgent,
		metadataTimeout: 30 * time.Second,
		targetTimeout:   10 * time.Minute,
	}
	transport.client = &http.Client{
		Transport:     roundTripper,
		CheckRedirect: transport.checkRedirect,
	}
	return transport, nil
}

func (t *Transport) DownloadFile(rawURL string, maxLength int64, _ time.Duration) ([]byte, error) {
	return t.Download(context.Background(), rawURL, maxLength)
}

func (t *Transport) Download(ctx context.Context, rawURL string, maxLength int64) ([]byte, error) {
	u, err := validateRemoteURL(rawURL)
	if err != nil {
		return nil, err
	}
	if _, ok := t.allowedHosts[u.Host]; !ok {
		return nil, fmt.Errorf("host %q is not allow-listed", u.Host)
	}
	if maxLength <= 0 || maxLength > TargetMaxLength {
		return nil, fmt.Errorf("download limit %d is outside the allowed range", maxLength)
	}
	timeout := t.metadataTimeout
	if maxLength > TargetsMaxLength {
		timeout = t.targetTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		data, retry, err := t.downloadOnce(ctx, u, maxLength)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

func (t *Transport) downloadOnce(ctx context.Context, u *url.URL, maxLength int64) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, false, err
	}
	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	response, err := t.client.Do(req)
	if err != nil {
		return nil, ctx.Err() == nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		retry := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return nil, retry, &metadata.ErrDownloadHTTP{StatusCode: response.StatusCode, URL: u.String()}
	}
	if header := response.Header.Get("Content-Length"); header != "" {
		length, parseErr := strconv.ParseInt(header, 10, 64)
		if parseErr != nil {
			return nil, false, fmt.Errorf("invalid Content-Length: %w", parseErr)
		}
		if length > maxLength {
			return nil, false, &metadata.ErrDownloadLengthMismatch{Msg: fmt.Sprintf("response length %d exceeds limit %d", length, maxLength)}
		}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxLength+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > maxLength {
		return nil, false, &metadata.ErrDownloadLengthMismatch{Msg: fmt.Sprintf("response length exceeds limit %d", maxLength)}
	}
	return data, false, nil
}

func (t *Transport) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > MaxRedirects {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "https" || len(via) == 0 || !strings.EqualFold(req.URL.Host, via[0].URL.Host) {
		return errors.New("redirect must remain on the original HTTPS host")
	}
	_, err := validateRemoteURL(req.URL.String())
	return err
}

func validateRemoteURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("remote URL must be absolute HTTPS without user information or fragment")
	}
	decoded, err := url.PathUnescape(u.EscapedPath())
	if err != nil {
		return nil, errors.New("remote URL has invalid escaping")
	}
	if strings.Contains(decoded, "\\") || strings.ContainsRune(decoded, '\x00') {
		return nil, errors.New("remote URL has an ambiguous path")
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return nil, errors.New("remote URL path traversal is not allowed")
		}
	}
	if decoded != "" && path.Clean(decoded) != decoded && path.Clean(decoded)+"/" != decoded {
		return nil, errors.New("remote URL path is not canonical")
	}
	return u, nil
}
