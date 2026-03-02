package http

import (
	"bytes"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
)

var (
	methodStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	urlStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	headerKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	headerVal   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	statusOk    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	statusRedir = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	statusErr   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	arrowSend   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("▶")
	arrowRecv   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("◀")
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	divider     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", 60))
)

func debugRequest(req *nethttp.Request, body []byte) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, divider)
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		arrowSend,
		methodStyle.Render(req.Method),
		urlStyle.Render(req.URL.String()),
	)
	fmt.Fprintln(os.Stderr, divider)

	printHeaders(req.Header, arrowSend)

	if len(body) > 0 {
		fmt.Fprintln(os.Stderr)
		ct := req.Header.Get("Content-Type")
		printBody(body, ct)
	}
}

func debugResponse(resp *nethttp.Response) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, divider)
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		arrowRecv,
		dimStyle.Render(resp.Proto),
		styledStatus(resp.StatusCode, resp.Status),
	)
	fmt.Fprintln(os.Stderr, divider)

	printHeaders(resp.Header, arrowRecv)

	if resp.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		if err == nil && len(raw) > 0 {
			resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), resp.Body))
			fmt.Fprintln(os.Stderr)
			ct := resp.Header.Get("Content-Type")
			printBody(raw, ct)
		}
	}
	fmt.Fprintln(os.Stderr)
}

func printHeaders(h nethttp.Header, _ string) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(os.Stderr, "  %s %s\n",
			headerKey.Render(k+":"),
			headerVal.Render(strings.Join(h[k], ", ")),
		)
	}
}

func styledStatus(code int, status string) string {
	switch {
	case code >= 200 && code < 300:
		return statusOk.Render(status)
	case code >= 300 && code < 400:
		return statusRedir.Render(status)
	default:
		return statusErr.Render(status)
	}
}

func printBody(body []byte, contentType string) {
	lexer := detectLexer(contentType, body)
	if lexer != "" {
		var buf bytes.Buffer
		err := quick.Highlight(&buf, string(body), lexer, "terminal256", "monokai")
		if err == nil {
			fmt.Fprint(os.Stderr, buf.String())
			return
		}
	}
	fmt.Fprint(os.Stderr, dimStyle.Render(string(body)))
}

func detectLexer(contentType string, body []byte) string {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		return "json"
	case strings.Contains(ct, "xml"):
		return "xml"
	case strings.Contains(ct, "html"):
		return "html"
	case strings.Contains(ct, "javascript"):
		return "javascript"
	case strings.Contains(ct, "css"):
		return "css"
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 {
		switch {
		case trimmed[0] == '{' || trimmed[0] == '[':
			return "json"
		case bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<soap")):
			return "xml"
		case bytes.HasPrefix(trimmed, []byte("<!DOCTYPE")) || bytes.HasPrefix(trimmed, []byte("<html")):
			return "html"
		}
	}
	return ""
}
