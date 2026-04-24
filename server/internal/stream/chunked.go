package stream

import (
	"fmt"
	"io"
	"net/http"

	"github.com/victor/coros-music/internal/httpx"
)

const Boundary = "corospart"

// Writer wraps an http.ResponseWriter to emit multipart/mixed parts.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	total   int
	index   int
}

// NewWriter initializes multipart headers and returns a Writer.
func NewWriter(w http.ResponseWriter, total int) *Writer {
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", Boundary))
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	f, _ := w.(http.Flusher)

	return &Writer{w: w, flusher: f, total: total, index: 0}
}

// WritePart writes a single MP3 part with appropriate headers.
func (mw *Writer) WritePart(filename string, data []byte) error {
	mw.index++

	header := fmt.Sprintf("\r\n--%s\r\nContent-Disposition: %s\r\nX-Track-Index: %d\r\nX-Track-Total: %d\r\n\r\n",
		Boundary, httpx.FormatAttachmentContentDisposition(filename), mw.index, mw.total)

	if _, err := io.WriteString(mw.w, header); err != nil {
		return fmt.Errorf("write part header: %w", err)
	}

	if _, err := mw.w.Write(data); err != nil {
		return fmt.Errorf("write part body: %w", err)
	}

	if mw.flusher != nil {
		mw.flusher.Flush()
	}

	return nil
}

// WriteErrorPart writes an error part for a track that failed.
func (mw *Writer) WriteErrorPart(filename, errMsg string) error {
	mw.index++

	header := fmt.Sprintf("\r\n--%s\r\nContent-Disposition: %s\r\nX-Track-Index: %d\r\nX-Track-Total: %d\r\nX-Track-Error: %s\r\n\r\n",
		Boundary, httpx.FormatAttachmentContentDisposition(filename), mw.index, mw.total, errMsg)

	if _, err := io.WriteString(mw.w, header); err != nil {
		return fmt.Errorf("write error part: %w", err)
	}

	if mw.flusher != nil {
		mw.flusher.Flush()
	}

	return nil
}

// Close writes the closing boundary.
func (mw *Writer) Close() error {
	closing := fmt.Sprintf("\r\n--%s--\r\n", Boundary)
	_, err := io.WriteString(mw.w, closing)
	if mw.flusher != nil {
		mw.flusher.Flush()
	}
	return err
}
