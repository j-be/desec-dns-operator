package desec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	Domain string
	token  string

	mgmtHost     string
	updateIpHost string
}

func (c Client) getMgmtBaseUrl() string {
	return c.mgmtHost + "/api/v1/domains/"
}

func (c Client) getUpdateIpBaseUrl() string {
	return c.updateIpHost
}

func NewClient(domain string, configDir string) (Client, error) {
	token, err := os.ReadFile(configDir + "/secret/token")
	if err != nil {
		return Client{}, err
	}
	mgmtHost, err := os.ReadFile(configDir + "/config/mgmtHost")
	if err != nil {
		mgmtHost = []byte("https://desec.io")
	}
	updateIpHost, err := os.ReadFile(configDir + "/config/updateIpHost")
	if err != nil {
		updateIpHost = []byte("https://update.dedyn.io")
	}

	return Client{
		Domain: domain,
		token:  string(token),

		mgmtHost:     string(mgmtHost),
		updateIpHost: string(updateIpHost),
	}, nil
}

func get[T any](url string, token string, dest *T) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadJson))
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

	return json.NewDecoder(res.Body).Decode(dest)
}

func (c Client) GetDomains() ([]Domain, error) {
	domains := make([]Domain, 0)
	err := get(c.getMgmtBaseUrl(), c.token, &domains)
	return domains, err
}

func (c Client) GetRRSets() ([]RRSet, error) {
	rrsets := make([]RRSet, 0)
	err := get(c.getMgmtBaseUrl()+c.Domain+"/rrsets/", c.token, &rrsets)
	return rrsets, err
}

func (c Client) CreateRRSet(rrset RRSet) (RRSet, error) {
	dest := RRSet{}
	err := post(c.getMgmtBaseUrl()+c.Domain+"/rrsets/", c.token, rrset, &dest)
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
	err := post(c.getMgmtBaseUrl(), c.token, createDomainPayload{Name: c.Domain}, &dest)
	return dest, err
}

func (c Client) UpdateIp(ips []string) error {
	url := fmt.Sprintf(
		"%s?hostname=%s&myip=%s",
		c.getUpdateIpBaseUrl(),
		url.QueryEscape(c.Domain),
		url.QueryEscape(strings.Join(ips, ",")),
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Token "+c.token)
	resp, err := http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode != 200 {
		return fmt.Errorf("got status code %d", resp.StatusCode)
	}

	return err
}
