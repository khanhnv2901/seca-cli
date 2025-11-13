package cmd

import "testing"

func TestMakeVerificationCommand(t *testing.T) {
	builder := makeVerificationCommand("results.json")
	cmd := builder(HashAlgorithmSHA512)
	expected := "sha512sum -c audit.csv.sha512 && sha512sum -c results.json.sha512"
	if cmd != expected {
		t.Fatalf("expected %s, got %s", expected, cmd)
	}

	builder = makeVerificationCommand("records.json")
	cmd = builder(HashAlgorithmSHA256)
	expected = "sha256sum -c audit.csv.sha256 && sha256sum -c records.json.sha256"
	if cmd != expected {
		t.Fatalf("expected %s, got %s", expected, cmd)
	}
}
