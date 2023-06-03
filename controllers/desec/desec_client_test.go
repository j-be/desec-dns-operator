package desec

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const mockDomain = `{"created":"2023-06-02T18:28:14.745468Z","published":"2023-06-02T18:29:36.137504Z","name":"some-domain.dedyn.io","minimum_ttl":60,"touched":"2023-06-02T18:46:49.868553Z"}`
const mockDomains = "[" + mockDomain + `,
	{"created":"2022-08-07T14:00:36.964836Z","published":"2022-08-07T14:18:41.911883Z","name":"some-other-domain.dedyn.io","minimum_ttl":60,"touched":"2022-08-07T14:18:41.911883Z"},
	{"created":"2017-08-01T17:49:07Z","published":"2022-06-18T02:37:46.501306Z","name":"another-domain.dedyn.io","minimum_ttl":60,"touched":"2023-06-03T07:50:34.194281Z"}
]`

const mockCname = `{"created":"2022-06-18T02:19:08.709359Z","domain":"some-domain.dedyn.io","subname":"www","name":"www.some-domain.dedyn.io.","records":["some-domain.dedyn.io."],"ttl":3600,"type":"CNAME","touched":"2022-06-18T02:19:08.713992Z"}`
const mockRrsets = "[" + mockCname + `,
	{"created":"2017-12-07T10:20:12.727000Z","domain":"some-domain.dedyn.io","subname":"","name":"some-domain.dedyn.io.","records":["1.2.3.4"],"ttl":60,"type":"A","touched":"2023-06-03T08:21:34.591942Z"},
	{"created":"2017-11-07T14:17:29.284000Z","domain":"some-domain.dedyn.io","subname":"","name":"some-domain.dedyn.io.","records":["ns1.desec.io.","ns2.desec.org."],"ttl":60,"type":"NS","touched":"2017-11-07T14:17:29.284000Z"}
]`

func createMgmtClient(server *httptest.Server) Client {
	host := server.Listener.Addr().String()
	return Client{
		Domain:   "some-domain.dedyn.io",
		scheme:   "http",
		token:    "I'm a token",
		mgmtHost: host,
	}
}

func createUpdateIpClient(server *httptest.Server) Client {
	host := server.Listener.Addr().String()
	return Client{
		Domain:       "some-domain.dedyn.io",
		scheme:       "http",
		token:        "I'm a token",
		updateIpHost: host,
	}
}

func TestGetDomains(t *testing.T) {
	t.Run("TestBasic", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v1/domains/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			_, err := w.Write([]byte(mockDomains))
			assert.NoError(t, err)
		}))
		defer server.Close()
		var client = createMgmtClient(server)
		// When
		domains, err := client.GetDomains()
		// Then
		assert.NoError(t, err)
		assert.Len(t, domains, 3)
		someDomain := domains[0]
		assert.Equal(t, "some-domain.dedyn.io", someDomain.Name)
		assert.Equal(t, int64(60), someDomain.Minimum_TTL)
	})
}

func TestGetRrset(t *testing.T) {
	t.Run("TestBasic", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v1/domains/some-domain.dedyn.io/rrsets/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			_, err := w.Write([]byte(mockRrsets))
			assert.NoError(t, err)
		}))
		defer server.Close()
		var client = createMgmtClient(server)
		// When
		rrsets, err := client.GetRRSets()
		// Then
		assert.NoError(t, err)
		assert.Len(t, rrsets, 3)
		{
			cname := rrsets[0]
			assert.Equal(t, "some-domain.dedyn.io", cname.Domain)
			assert.Equal(t, "www", cname.Subname)
			assert.Equal(t, "www.some-domain.dedyn.io.", cname.Name)
			assert.Equal(t, "CNAME", cname.Type)
			assert.Equal(t, []string{"some-domain.dedyn.io."}, cname.Records)
			assert.Equal(t, int64(3600), cname.TTL)
		}

		{
			a := rrsets[1]
			assert.Equal(t, "some-domain.dedyn.io", a.Domain)
			assert.Empty(t, a.Subname)
			assert.Equal(t, "some-domain.dedyn.io.", a.Name)
			assert.Equal(t, "A", a.Type)
			assert.Equal(t, []string{"1.2.3.4"}, a.Records)
			assert.Equal(t, int64(60), a.TTL)
		}

		{
			ns := rrsets[2]
			assert.Equal(t, "some-domain.dedyn.io", ns.Domain)
			assert.Empty(t, ns.Subname)
			assert.Equal(t, "some-domain.dedyn.io.", ns.Name)
			assert.Equal(t, "NS", ns.Type)
			assert.Equal(t, []string{"ns1.desec.io.", "ns2.desec.org."}, ns.Records)
			assert.Equal(t, int64(60), ns.TTL)
		}
	})
}

func TestCreateCNAME(t *testing.T) {
	t.Run("TestBasic", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v1/domains/some-domain.dedyn.io/rrsets/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Contains(t,
				string(body),
				`"domain":"some-domain.dedyn.io","subname":"www","name":"www.some-domain.dedyn.io.","type":"CNAME","records":["some-domain.dedyn.io."]`,
			)
			w.WriteHeader(201)
			_, err = w.Write([]byte(mockCname))
			assert.NoError(t, err)
		}))
		defer server.Close()
		var client = createMgmtClient(server)
		// When
		cname, err := client.CreateCNAME("www")
		// Then
		assert.NoError(t, err)
		assert.Equal(t, "some-domain.dedyn.io", cname.Domain)
		assert.Equal(t, "www", cname.Subname)
		assert.Equal(t, "www.some-domain.dedyn.io.", cname.Name)
		assert.Equal(t, "CNAME", cname.Type)
		assert.Equal(t, []string{"some-domain.dedyn.io."}, cname.Records)
		assert.Equal(t, int64(3600), cname.TTL)
	})
}

func TestCreateDomain(t *testing.T) {
	t.Run("TestBasic", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v1/domains/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			w.WriteHeader(201)
			_, err := w.Write([]byte(mockDomain))
			assert.NoError(t, err)
		}))
		defer server.Close()
		var client = createMgmtClient(server)
		// When
		domain, err := client.CreateDomain()
		// Then
		assert.NoError(t, err)
		assert.Equal(t, "some-domain.dedyn.io", domain.Name)
		assert.Equal(t, int64(60), domain.Minimum_TTL)
	})
}

func TestUpdateIp(t *testing.T) {
	t.Run("TestBasic", func(t *testing.T) {
		// Given
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/", r.URL.Path)
			assert.Equal(t, "Token I'm a token", r.Header.Get("Authorization"))
			assert.Equal(t, "1.2.3.4,2.3.4.5", r.URL.Query().Get("myip"))
			assert.Equal(t, "some-domain.dedyn.io", r.URL.Query().Get("hostname"))
			_, err := w.Write([]byte("good"))
			assert.NoError(t, err)
		}))
		defer server.Close()
		var client = createUpdateIpClient(server)
		// When
		err := client.UpdateIp([]string{"1.2.3.4", "2.3.4.5"})
		// Then
		assert.NoError(t, err)
	})
}
