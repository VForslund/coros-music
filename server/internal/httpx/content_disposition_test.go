package httpx

import (
	"strings"
	"testing"
)

func TestFormatAttachmentContentDispositionUsesRFC5987ForUnicode(t *testing.T) {
	got := FormatAttachmentContentDisposition("15ZsviIr4apvdoQUup7yFn — Civil War - Hjältar ifrån Dalarna.mp3")

	if strings.Contains(got, "Hjältar") || strings.Contains(got, "—") {
		t.Fatalf("expected raw header to stay ASCII-safe, got %q", got)
	}

	wantFallback := `filename="15ZsviIr4apvdoQUup7yFn _ Civil War - Hj_ltar ifr_n Dalarna.mp3"`
	if !strings.Contains(got, wantFallback) {
		t.Fatalf("expected ASCII fallback %q in %q", wantFallback, got)
	}

	wantRFC5987 := "filename*=UTF-8''15ZsviIr4apvdoQUup7yFn%20%E2%80%94%20Civil%20War%20-%20Hj%C3%A4ltar%20ifr%C3%A5n%20Dalarna.mp3"
	if !strings.Contains(got, wantRFC5987) {
		t.Fatalf("expected UTF-8 filename* %q in %q", wantRFC5987, got)
	}
}

func TestFormatAttachmentContentDispositionKeepsASCIIFilenamesReadable(t *testing.T) {
	got := FormatAttachmentContentDisposition("playlist.zip")

	if !strings.Contains(got, `filename="playlist.zip"`) {
		t.Fatalf("expected quoted ASCII filename in %q", got)
	}

	if !strings.Contains(got, "filename*=UTF-8''playlist.zip") {
		t.Fatalf("expected filename* fallback in %q", got)
	}
}
