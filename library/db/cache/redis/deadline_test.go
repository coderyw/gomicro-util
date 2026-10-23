package redis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// Exercise the actual production constructor, including the cluster TLS branch.
// miniredis advertises itself as a single node owning all cluster slots.
func newDeadlineTestRedis(t *testing.T, cluster bool) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.NewMiniRedis()
	if cluster {
		certServer := httptest.NewTLSServer(http.NotFoundHandler())
		config := certServer.TLS.Clone()
		certServer.Close()
		if err := mr.StartTLS(config); err != nil {
			t.Fatal(err)
		}
	} else if err := mr.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	name := t.Name()
	if err := initOneRedis([]string{mr.Addr()}, "", 0, cluster, 2, 1, false, name); err != nil {
		t.Fatal(err)
	}
	r := redisClientMap[name]
	t.Cleanup(func() { _ = r.UniversalClient.Close(); delete(redisClientMap, name) })
	return r, mr
}

// Preserve the pre-existing timeout policy for both production construction paths.
func TestRedisClientTimeoutPolicyCompatibility(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		name := "standalone"
		if cluster {
			name = "cluster"
		}
		t.Run(name, func(t *testing.T) {
			r, _ := newDeadlineTestRedis(t, cluster)
			switch client := r.UniversalClient.(type) {
			case *goredis.Client:
				if client.Options().ContextTimeoutEnabled {
					t.Fatal("global context timeouts unexpectedly enabled")
				}
			case *goredis.ClusterClient:
				if client.Options().ContextTimeoutEnabled {
					t.Fatal("global context timeouts unexpectedly enabled")
				}
			default:
				t.Fatal("unexpected client type")
			}
			if err := r.UniversalClient.Set(context.Background(), "compatibility-key", "VALUE", 0).Err(); err != nil {
				t.Fatal(err)
			}
			if got, err := r.UniversalClient.Get(context.Background(), "compatibility-key").Result(); err != nil || got != "VALUE" {
				t.Fatalf("ordinary command changed: %q %v", got, err)
			}
		})
	}
}
