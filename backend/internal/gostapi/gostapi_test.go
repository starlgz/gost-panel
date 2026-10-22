package gostapi

import "testing"

func TestSavedConfigKeepsAPIEndpoint(t *testing.T) {
	for _, endpoint := range []string{"127.0.0.1:18080", "127.0.0.1:19081", "[::1]:18080"} {
		client := New("http://"+endpoint, "test-user", "test-only", "")
		config := client.buildFullConfig(nil)
		api := config["api"].(map[string]any)
		if api["addr"] != endpoint {
			t.Fatalf("API listen address=%v; want %s", api["addr"], endpoint)
		}
	}
}
