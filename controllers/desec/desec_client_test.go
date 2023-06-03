package desec

import (
	"fmt"
	"testing"

	"github.com/j-be/desec-dns-operator/controllers/util"
	"github.com/stretchr/testify/assert"
)

var client = Client{
	Domain: "great-horned-owl.dedyn.io",
	token:  util.TOKEN,
}

func TestGetDomains(t *testing.T) {
	domains, err := client.GetDomains()
	assert.NoError(t, err)
	fmt.Println(domains)
}

func TestGetRrset(t *testing.T) {
	rrsets, err := client.GetRRSets()
	assert.NoError(t, err)
	fmt.Println(rrsets)
}
