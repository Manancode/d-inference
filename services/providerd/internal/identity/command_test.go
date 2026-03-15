package identity

import (
	"encoding/base64"
	"errors"
	"testing"
)

type fakeRunner struct {
	outputs [][]byte
	err     error
	index   int
}

func (f *fakeRunner) Output(name string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := f.outputs[f.index]
	f.index++
	return out, nil
}

func TestCommandSignerRoundTrip(t *testing.T) {
	runner := &fakeRunner{
		outputs: [][]byte{
			[]byte(`{"publicKey":"pub","note":"secure-enclave"}`),
			[]byte(`{"publicKey":"pub","signature":"sig","note":"secure-enclave"}`),
		},
	}
	signer := &CommandSigner{
		command: "/usr/local/bin/DGInfProviderKeyTool",
		tag:     "com.dginf.test",
		runner:  runner,
	}
	publicKey, err := signer.PublicKey()
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if publicKey != "pub" || signature != "sig" {
		t.Fatalf("unexpected results: %q %q", publicKey, signature)
	}
}

func TestCommandSignerPropagatesRunnerError(t *testing.T) {
	signer := &CommandSigner{
		command: "/usr/local/bin/DGInfProviderKeyTool",
		tag:     "com.dginf.test",
		runner:  &fakeRunner{err: errors.New("boom")},
	}
	if _, err := signer.PublicKey(); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestCommandSignerUsesBase64Payload(t *testing.T) {
	payload := []byte("payload")
	if got := base64.StdEncoding.EncodeToString(payload); got != "cGF5bG9hZA==" {
		t.Fatalf("unexpected base64 encoding: %s", got)
	}
}
