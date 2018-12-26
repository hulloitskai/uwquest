package uwquest_test

import (
	"testing"
)

func TestClient_Terms(t *testing.T) {
	terms, err := client.Terms()
	if err != nil {
		t.Fatalf("Error while fetching terms data: %v", err)
	}

	if n := len(terms); n == 0 {
		t.Fatal("Did not find any terms.")
	}

	t.Logf("Got terms: %v", terms)
}
