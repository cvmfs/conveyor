package jobdb

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	_ "github.com/lib/pq" // Import and register the PostgreSQL driver
	"github.com/pkg/errors"
)

// BackendConfig - database backend configuration for the job db service
type BackendConfig struct {
	Type     string
	Database string
	Username string
	Password string
	Host     string
	Port     int
}

// Backend - encapsulates the backend state
type Backend struct {
	db *sql.DB
}

// Close - closes the database connection
func (b *Backend) Close() {
	b.db.Close()
}

// GetJob - returns the ro
func (b *Backend) GetJob(id string, full bool) (string, error) {
	rows, err := b.db.Query("select * from Jobs where ID = $1", id)
	if err != nil {
		return "", errors.Wrap(err, "select query failed")
	}
	defer rows.Close()

	if !rows.Next() {
		return "", nil
	}

	var st job.Processed
	if err := rows.Scan(
		&st.ID, &st.Repository, &st.Payload, &st.RepositoryPath,
		&st.Script, &st.ScriptArgs, &st.RemoteScript,
		&st.Dependencies, &st.StartTime, &st.FinishTime,
		&st.Successful, &st.ErrorMessage); err != nil {
		return "", errors.Wrap(err, "scan failed")
	}

	v, err := json.Marshal(&st)
	if err != nil {
		return "", errors.Wrap(err, "JSON marshalling failed")
	}

	return string(v), nil
}

func startBackEnd(cfg BackendConfig) (*Backend, error) {
	db, err := sql.Open(cfg.Type, createDataSrcName(cfg))
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "connection ping failed")
	}

	return &Backend{db}, nil
}

func createDataSrcName(cfg BackendConfig) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%v dbname=%s sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}
