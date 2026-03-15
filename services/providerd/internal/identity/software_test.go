package identity

import "testing"

func TestSoftwareSignerRoundTrip(t *testing.T) {
	signer, err := NewSoftwareSigner()
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	publicKey, err := signer.PublicKey()
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	signature, err := signer.Sign([]byte("job-envelope"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	ok, err := Verify(publicKey, []byte("job-envelope"), signature)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !ok {
		t.Fatal("expected signature verification to succeed")
	}
}
