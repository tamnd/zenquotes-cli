package zenquotes

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.
// HTTP behaviour is covered in zenquotes_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "zenquotes" {
		t.Errorf("Scheme = %q, want zenquotes", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "zenquotes" {
		t.Errorf("Identity.Binary = %q, want zenquotes", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}

	typ, id, err := Domain{}.Classify("some-quote-id")
	if err != nil || typ != "quote" || id != "some-quote-id" {
		t.Errorf("Classify = (%q, %q, %v), want (quote, some-quote-id, nil)", typ, id, err)
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("quote", "123")
	want := "https://zenquotes.io"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}
