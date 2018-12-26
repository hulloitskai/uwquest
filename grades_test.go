package uwquest_test

import (
	"testing"
)

func TestClient_Grades(t *testing.T) {
	grades, err := client.Grades(0)
	if err != nil {
		t.Fatalf("Error fetching course grades")
	}

	if n := len(grades); n == 0 {
		t.Fatal("Did not find any grades.")
	}

	t.Logf("Got grades for term 0: %v", grades)
}
