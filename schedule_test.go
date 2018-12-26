package uwquest_test

import (
	"testing"
)

func TestClient_Schedules(t *testing.T) {
	s, err := client.Schedules(0)
	if err != nil {
		t.Fatalf("Error while fetching course schedule: %v", err)
	}

	if len(s) == 0 {
		t.Fatal("Did not find any course schedules for term 0.")
	}

	t.Logf("Got course schedules for term 0: %v\n", s)
}
