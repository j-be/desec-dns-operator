package desec

type AuditInfo struct {
	Created string `json:"created"`
	Touched string `json:"touched"`
}

type Domain struct {
	AuditInfo
	Name        string `json:"name"`
	Minimum_ttl int    `json:"minimum_ttl"`
	Published   string `json:"published"`
}

type RRSet struct {
	AuditInfo
	Domain  string   `json:"domain"`
	Subname string   `json:"subname"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int64    `json:"ttl"`
}

type createDomainPayload struct {
	Name string `json:"name"`
}
