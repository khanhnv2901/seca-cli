package cmd

import "testing"

func TestCanonicalTargetNormalizes(t *testing.T) {
	target := "example.com/path#section"
	got := canonicalTarget(target)
	want := "http://example.com/path"
	if got != want {
		t.Fatalf("canonicalTarget() = %s, want %s", got, want)
	}
}

func TestTargetSetAdd(t *testing.T) {
	set := newTargetSet()
	if !set.Add("https://example.com") {
		t.Fatal("expected first add to succeed")
	}
	if set.Add("https://example.com/") {
		t.Fatal("expected canonical duplicate to be rejected")
	}
	if set.Add("https://example.com/#frag") {
		t.Fatal("expected fragment duplicate to be rejected")
	}
	if !set.Add("https://example.com/login") {
		t.Fatal("expected unique path to be added")
	}
}
