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
	TenderID          string      `schema:"-"`
	TenderName        string      `schema:"name,required"`
	TenderDescription string      `schema:"description,required"`
	ServiceType       ServiceType `schema:"serviceType,required"`
	Status            Status      `schema:"-"`
	OrganizationID    string      `schema:"organizationId,required"`
	Version           int32       `schema:"-"` // min 1, def 1
	CreatedAt         string      `schema:"-"` // RFC3339 format.
	Author            string      `achema:"-"`
}

type TendersRepo interface {
	Check(username string) bool
	GetQuery(limit, offset int32, serviceType []ServiceType) ([]*Tender, error)
	GetByID(id string) (*Tender, error)
	GetMy(limit, offset int32, username string) ([]*Tender, error)
	Add(tender *Tender) (string, error)
	Update(newItem *Tender) (bool, error)
	Delete(id string) (bool, error)
}
