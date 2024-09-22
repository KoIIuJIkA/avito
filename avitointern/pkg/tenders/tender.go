package tenders

type ServiceType string

const (
	Construction ServiceType = "Construction"
	Delivery     ServiceType = "Delivery"
	Manufacture  ServiceType = "Manufacture"
)

type Status string

const (
	Created   Status = "Created"
	Published Status = "Published"
	Closed    Status = "Closed"
)

type Tender struct {
	TenderID          string               `json:"tenderID"`
	TenderName        string               `json:"TenderName"`
	TenderDescription string               `json:"TenderDescription"`
	ServiceType       ServiceType          `json:"ServiceType"`
	Status            Status               `json:"Status"`
	OrganizationID    string               `json:"OrganizationID"`
	Version           int32                `json:"Version"`   // min 1, def 1
	CreatedAt         string               `json:"CreatedAt"` // RFC3339 format.
	Author            string               `json:"Author"`
	Versions          map[int32]*TenderVer `json:"Versions"`
}

type TenderVer struct {
	TenderName        string `json:"name"`
	TenderDescription string `json:"description"`
	ServiceType       string `json:"serviceType"`
	Version           int32  `json:"Version"`
	Status            Status `json:"status"`
}

type TendersRepo interface {
	Check(username string) bool
	GetQuery(limit, offset int32, serviceType []ServiceType) ([]*Tender, error)
	GetByID(id string) (*Tender, error)
	GetMy(limit, offset int32, username string) ([]*Tender, error)
	Add(tender *Tender) (string, error)
}
