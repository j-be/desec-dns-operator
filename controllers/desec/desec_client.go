package desec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"
)

const desecBase = "https://desec.io/api/v1/domains/"
const updateIp = "https://update.dedyn.io"

type Client struct {
	Domain string
	Token  string
}

func get[T any](url string, token string, dest *T) error {
	req, err := http.NewRequest(http.MethodGet, desecBase+url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Token "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 404 {
		return nil
	}

	err = json.NewDecoder(res.Body).Decode(dest)
	if err != nil {
		return err
	}

	return nil
}

func post[T any, R any](url string, token string, payload R, dest *T) error {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, desecBase+url, bytes.NewReader(payloadJson))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 201 {
		return fmt.Errorf("got status %d while trying to POST %v", res.StatusCode, payload)
	}

	err = json.NewDecoder(res.Body).Decode(dest)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) GetOwnerOf(qname string) (*Domain, error) {
	owners := []Domain{}

	err := get("?owns_qname="+url.QueryEscape(qname), c.Token, &owners)
	if err != nil {
		return nil, err
	} else if len(owners) == 0 {
		return nil, nil
	}

	owner := owners[0]
	return &owner, nil
}

func (c Client) GetOwnersOf(qnames []string) ([]string, error) {
	domainOwner := []string{}
	for _, qname := range qnames {
		owner, _ := c.GetOwnerOf(qname)
		var ownerName string
		if owner == nil {
			ownerName = qname
		} else {
			ownerName = owner.Name
		}
		if !slices.Contains(domainOwner, ownerName) {
			domainOwner = append(domainOwner, ownerName)
		}
	}
	return domainOwner, nil
}

func (c Client) GetDomains() ([]Domain, error) {
	domains := []Domain{}

	err := get("", c.Token, &domains)
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func (c Client) GetRRSets() ([]RRSet, error) {
	rrsets := []RRSet{}

	err := get(c.Domain+"/rrsets/", c.Token, &rrsets)
	if err != nil {
		return nil, err
	}

	return rrsets, nil
}

func (c Client) CreateRRSet(rrset RRSet) (RRSet, error) {
	dest := RRSet{}
	err := post(rrset.Domain+"/rrsets/", c.Token, rrset, &dest)
	return dest, err
}

func (c Client) CreateCNAME(subname string) (RRSet, error) {
	return c.CreateRRSet(RRSet{
		Domain:  c.Domain,
		Subname: subname,
		Name:    subname + "." + c.Domain + ".",
		Type:    "CNAME",
		Records: []string{c.Domain + "."},
		TTL:     3600,
	})
}

func (c Client) CreateDomain() (Domain, error) {
	dest := Domain{}
	err := post("", c.Token, createDomainPayload{Name: c.Domain}, &dest)
	return dest, err
}

func (c Client) UpdateIp(ips []string) error {
	url := fmt.Sprintf("%s?hostname=%s&myip=%s", updateIp, url.QueryEscape(c.Domain), url.QueryEscape(strings.Join(ips, ",")))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Token "+c.Token)
	resp, err := http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode != 200 {
		return fmt.Errorf("got status code %d", resp.StatusCode)
	}

	return err
}
