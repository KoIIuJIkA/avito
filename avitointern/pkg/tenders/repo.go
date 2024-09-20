package tenders

import (
	"sync"
)

type TenderMemoryRepository struct {
	lastID string
	data   []*Tender
	mu     *sync.RWMutex
}

var _ TendersRepo = &TenderMemoryRepository{}

func NewMemoryRepo() *TenderMemoryRepository {
	return &TenderMemoryRepository{
		data: make([]*Tender, 0, 10),
		mu:   &sync.RWMutex{},
	}
}

func (repo *TenderMemoryRepository) Check(username string) bool {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	flag := false
	for _, v := range repo.data {
		if v.Author == username {
			flag = true
			return flag
		}
	}

	return flag
}

func (repo *TenderMemoryRepository) GetQuery(limit, offset int32, serviceType []ServiceType) ([]*Tender, error) {
	ContainsString := func(slice []ServiceType, value ServiceType) bool {
		for _, v := range slice {
			if v == value {
				return true
			}
		}
		return false
	}

	list := make([]*Tender, 0)

	repo.mu.RLock()
	defer repo.mu.RUnlock()

	for i := offset; i < int32(len(repo.data)) && i-offset < limit; i++ {
		if len(serviceType) == 0 || ContainsString(serviceType, repo.data[i].ServiceType) {
			list = append(list, repo.data[i])
		}
	}
	return list, nil
}

func (repo *TenderMemoryRepository) GetMy(limit, offset int32, username string) ([]*Tender, error) {
	list := make([]*Tender, 0)

	repo.mu.RLock()
	defer repo.mu.RUnlock()

	for i := offset; i < int32(len(repo.data)) && i-offset < limit; i++ {
		if username == repo.data[i].Author {
			list = append(list, repo.data[i])
		}
	}

	return list, nil
}

func (repo *TenderMemoryRepository) GetByID(id string) (*Tender, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	for _, tender := range repo.data {
		if tender.TenderID == id {
			return tender, nil
		}
	}
	return nil, nil
}

func (repo *TenderMemoryRepository) Add(tender *Tender) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.data = append(repo.data, tender)
	repo.lastID = tender.TenderID

	return repo.lastID, nil
}
