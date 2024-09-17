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
		if ContainsString(serviceType, repo.data[i].ServiceType) {
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

	return repo.lastID, nil
}

// WARNING! Template.
func (repo *TenderMemoryRepository) Update(newItem *Tender) (bool, error) {
	// for _, tender := range repo.data {
	// 	if tender.ID != newItem.ID {
	// 		continue
	// 	}
	// 	tender.Title = newItem.Title
	// 	tender.Description = newItem.Description
	// 	return true, nil
	// }
	return false, nil
}

// WARNING! Template.
func (repo *TenderMemoryRepository) Delete(id string) (bool, error) {
	// repo.mu.Lock()
	// defer repo.mu.RUnlock()

	// i := -1
	// for idx, tender := range repo.data {
	// 	if tender.TenderID != id {
	// 		continue
	// 	}
	// 	i = idx
	// }
	// if i < 0 {
	// 	return false, nil
	// }

	// if i < len(repo.data)-1 {
	// 	copy(repo.data[i:], repo.data[i+1:])
	// }
	// repo.data[len(repo.data)-1] = nil // or the zero value of T
	// repo.data = repo.data[:len(repo.data)-1]

	return true, nil
}
