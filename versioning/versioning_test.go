package versioning

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestApp(t *testing.T) {
	app := App{}
	go app.Run()

	{
		body := `{"firstName": "John", "lastName": "Sample"}`
		resp, err := http.Post(
			"http://localhost:8080/v1/greet",
			"application/json",
			strings.NewReader(body),
		)
		if err != nil {
			t.Fatal(err)
		}
		testGreeting(t, resp, "Hello John Sample!")
	}

	{
		body := `{"name":"Jane Sample"}`
		resp, err := http.Post(
			"http://localhost:8080/v2/greet",
			"application/json",
			strings.NewReader(body),
		)
		if err != nil {
			t.Fatal(err)
		}
		testGreeting(t, resp, "Hello Jane Sample!")
	}
}

func testGreeting(t *testing.T, resp *http.Response, want string) {
	var response struct {
		Greeting string `json:"greeting"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if response.Greeting != want {
		t.Errorf("wanted greeting %q. got=%q", want, response.Greeting)
	}
}
