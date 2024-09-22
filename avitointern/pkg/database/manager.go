package database

import (
	"avitointern/pkg/tenders"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

type SQLManager struct {
	DB *sql.DB
}

type Database interface {
	Init()
	Close()
	InsertTender(tender *tenders.Tender) (string, error)
	GetTenderByID(tenderID string) (*tenders.Tender, error)
	GetQuery(limit, offset int32, serviceTypes []tenders.ServiceType) ([]*tenders.Tender, error)
	My(limit, offset int32, author string) ([]*tenders.Tender, error)
	UpdateTenderStatus(tenderID string, newStatus tenders.Status) (*tenders.Tender, error)
	EditTender(tenderID string, name, description string, serviceType tenders.ServiceType) (*tenders.Tender, error)
	Rollback(tenderID string, version int32) (*tenders.TenderVer, error)
}

var _ Database = &SQLManager{}

func NewMemoryRepo() *SQLManager {
	return &SQLManager{}
}

func (m *SQLManager) Init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with environment variables")
	}
	serverAddress := os.Getenv("SERVER_ADDRESS")
	postgresConn := os.Getenv("POSTGRES_CONN")
	postgresUsername := os.Getenv("POSTGRES_USERNAME")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDatabase := os.Getenv("POSTGRES_DATABASE")

	fmt.Println(serverAddress, postgresConn)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUsername, postgresPassword, postgresHost, postgresPort, postgresDatabase)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	m.DB = db
	log.Println("Successfully connected to the database!")
}

func (m *SQLManager) Close() {
	if err := m.DB.Close(); err != nil {
		log.Println("Error closing database connection:", err)
	}
}

func (m *SQLManager) InsertTender(tender *tenders.Tender) (string, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	query := `INSERT INTO tenders (tender_id, tender_name, tender_description, 
				service_type, status, organization_id, version, created_at, author)
			  	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = tx.Exec(query,
		tender.TenderID, tender.TenderName, tender.TenderDescription,
		tender.ServiceType, tender.Status, tender.OrganizationID,
		tender.Version, tender.CreatedAt, tender.Author)
	if err != nil {
		return "", err
	}

	query = `INSERT INTO tender_versions (tender_id, version, tender_name, tender_description, service_type, status)
			 VALUES ($1, $2, $3, $4, $5, $6)`
	for _, version := range tender.Versions {
		_, err = tx.Exec(query, tender.TenderID, version.Version, version.TenderName,
			version.TenderDescription, version.ServiceType, version.Status)
		if err != nil {
			return "", err
		}
	}

	if err = tx.Commit(); err != nil {
		log.Println("err in tx.commit")
		return "", err
	}

	return tender.TenderID, nil
}

func (m *SQLManager) GetTenderByID(tenderID string) (*tenders.Tender, error) {
	query := `SELECT tender_id, tender_name, tender_description, service_type, status, organization_id, version, created_at, author FROM tenders WHERE tender_id = $1`

	var tender tenders.Tender
	err := m.DB.QueryRow(query, tenderID).Scan(&tender.TenderID, &tender.TenderName, &tender.TenderDescription, &tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt, &tender.Author)
	if err != nil {
		return nil, err
	}

	queryVersions := `SELECT version, tender_name, tender_description, service_type FROM tender_versions WHERE tender_id = $1`
	rows, err := m.DB.Query(queryVersions, tenderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tender.Versions = make(map[int32]*tenders.TenderVer)
	for rows.Next() {
		var version tenders.TenderVer
		err = rows.Scan(&version.Version, &version.TenderName, &version.TenderDescription, &version.ServiceType)
		if err != nil {
			return nil, err
		}
		tender.Versions[version.Version] = &version
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &tender, nil
}

func (m *SQLManager) GetQuery(limit, offset int32, serviceTypes []tenders.ServiceType) ([]*tenders.Tender, error) {
	var query string
	var args []interface{}

	query = `SELECT tender_id, tender_name, tender_description, service_type, status, organization_id, version, created_at, author 
			 FROM tenders`

	if len(serviceTypes) > 0 {
		query += " WHERE service_type IN ("
		for i, service := range serviceTypes {
			if i > 0 {
				query += ", "
			}
			query += fmt.Sprintf("$%d", len(args)+1)
			args = append(args, service)
		}
		query += ")"
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tendersList []*tenders.Tender
	for rows.Next() {
		var tender tenders.Tender
		err = rows.Scan(&tender.TenderID, &tender.TenderName, &tender.TenderDescription,
			&tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt, &tender.Author)
		if err != nil {
			return nil, err
		}
		tendersList = append(tendersList, &tender)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tendersList, nil
}

func (m *SQLManager) My(limit, offset int32, author string) ([]*tenders.Tender, error) {
	var query string
	var args []interface{}

	query = `SELECT tender_id, tender_name, tender_description, service_type, status, organization_id, version, created_at, author 
			 FROM tenders`

	if author != "" {
		query += " WHERE author = $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, author)
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tendersList []*tenders.Tender
	for rows.Next() {
		var tender tenders.Tender
		err = rows.Scan(&tender.TenderID, &tender.TenderName, &tender.TenderDescription,
			&tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt, &tender.Author)
		if err != nil {
			return nil, err
		}
		tendersList = append(tendersList, &tender)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tendersList, nil
}

func (m *SQLManager) UpdateTenderStatus(tenderID string, newStatus tenders.Status) (*tenders.Tender, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	var tender tenders.Tender
	const querySel = `SELECT tender_id, tender_name, tender_description, service_type, status, organization_id, version, created_at, author
              FROM tenders WHERE tender_id = $1`
	err = tx.QueryRow(querySel, tenderID).Scan(&tender.TenderID, &tender.TenderName, &tender.TenderDescription,
		&tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt, &tender.Author)
	if err != nil {
		log.Println("tx.QueryRow with select 1")
		return nil, err
	}
	tender.Status = newStatus

	const query = `SELECT COUNT(*) 
			   FROM tender_versions WHERE tender_id = $1`
	var len int
	err = tx.QueryRow(query, tenderID).Scan(&len)
	if err != nil {
		log.Println("tx.QueryRow with select 2")
		return nil, err
	}
	newVersion := len + 1

	updateTenderQuery := `UPDATE tenders SET status = $1, version = $2 WHERE tender_id = $3`
	_, err = tx.Exec(updateTenderQuery, newStatus, newVersion, tender.TenderID)
	if err != nil {
		log.Println("tx.Exec with updateTenderQuery")
		return nil, err
	}

	const insertVersionQuery = `INSERT INTO tender_versions (tender_id, version, tender_name, tender_description, service_type, status)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(insertVersionQuery, tender.TenderID, newVersion, tender.TenderName,
		tender.TenderDescription, tender.ServiceType, newStatus)
	if err != nil {
		log.Println("tx.Exec with insertVersionQuery")
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		log.Println("err in tx.commit")
		return nil, err
	}

	return &tender, nil
}

func (m *SQLManager) EditTender(tenderID string, name, description string, serviceType tenders.ServiceType) (*tenders.Tender, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	var tender tenders.Tender
	query := `SELECT tender_id, tender_name, tender_description, service_type, status, organization_id, version, created_at, author
              FROM tenders WHERE tender_id = $1`
	err = tx.QueryRow(query, tenderID).Scan(&tender.TenderID, &tender.TenderName, &tender.TenderDescription,
		&tender.ServiceType, &tender.Status, &tender.OrganizationID, &tender.Version, &tender.CreatedAt, &tender.Author)
	if err != nil {
		return nil, err
	}
	tender.TenderName = name
	tender.TenderDescription = description
	tender.ServiceType = serviceType

	query = `SELECT COUNT(*) 
			   FROM tender_versions WHERE tender_id = $1`
	var len int
	err = tx.QueryRow(query, tenderID).Scan(&len)
	if err != nil {
		return nil, err
	}
	newVersion := len + 1

	updateTenderQuery := `UPDATE tenders SET version = $1, tender_name = $2, tender_description = $3, service_type = $4 WHERE tender_id = $5`
	_, err = tx.Exec(updateTenderQuery, newVersion, tender.TenderName, tender.TenderDescription, tender.ServiceType, tender.TenderID)
	if err != nil {
		return nil, err
	}

	insertVersionQuery := `INSERT INTO tender_versions (tender_id, version, tender_name, tender_description, service_type, status)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(insertVersionQuery, tender.TenderID, newVersion, tender.TenderName,
		tender.TenderDescription, tender.ServiceType, tender.Status)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		log.Println("err in tx.commit")
		return nil, err
	}

	return &tender, nil
}

func (m *SQLManager) Rollback(tenderID string, version int32) (*tenders.TenderVer, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	var tender tenders.TenderVer
	query := `SELECT tender_name, tender_description, service_type, status
              FROM tender_versions WHERE tender_id = $1`
	err = tx.QueryRow(query, tenderID).Scan(&tender.TenderName, &tender.TenderDescription,
		&tender.ServiceType, &tender.Status)
	if err != nil {
		return nil, err
	}

	query = `SELECT COUNT(*) 
			   FROM tender_versions WHERE tender_id = $1`
	var len int
	err = tx.QueryRow(query, tenderID).Scan(&len)
	if err != nil {
		return nil, err
	}
	newVersion := len + 1

	updateTenderQuery := `UPDATE tenders SET version = $1, tender_name = $2, tender_description = $3, service_type = $4 WHERE tender_id = $5`
	_, err = tx.Exec(updateTenderQuery, newVersion, tender.TenderName, tender.TenderDescription, tender.ServiceType, tenderID)
	if err != nil {
		return nil, err
	}

	insertVersionQuery := `INSERT INTO tender_versions (tender_id, version, tender_name, tender_description, service_type, status)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(insertVersionQuery, tenderID, newVersion, tender.TenderName,
		tender.TenderDescription, tender.ServiceType, tender.Status)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		log.Println("err in tx.commit")
		return nil, err
	}

	return &tender, nil
}
