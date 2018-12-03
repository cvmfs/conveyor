package cvmfs

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql" // Import and register the MySQL driver

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

// StartBackEnd initializes the backend of the job server
func StartBackEnd(cfg BackendConfig) (*Backend, error) {
	db, err := sql.Open(cfg.Type, createDataSrcName(cfg))
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "connection ping failed")
	}

	return &Backend{db}, nil
}

// Close - closes the database connection
func (b *Backend) Close() {
	b.db.Close()
}

// GetJobs - returns the rows from the job DB corresponding to the IDs
func (b *Backend) GetJobs(ids []string, full bool) (*GetJobReply, error) {
	reply := GetJobReply{Status: "ok", Reason: ""}

	queryStr := "select * from Jobs where Jobs.ID in ("
	params := make([]interface{}, len(ids))
	for i, v := range ids[0 : len(ids)-1] {
		queryStr += "?, "
		params[i] = v
	}
	queryStr += "?);"
	params[len(ids)-1] = ids[len(ids)-1]

	rows, err := b.db.Query(queryStr, params...)
	if err != nil {
		reason := "SQL query failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}
	defer rows.Close()

	for rows.Next() {
		st, err := scanRow(rows)
		if err != nil {
			reason := "SQL query scan failed"
			reply.Status = "error"
			reply.Reason = reason
			reply.IDs = []JobStatus{}
			reply.Jobs = []ProcessedJob{}
			return &reply, errors.Wrap(err, reason)
		}

		if full {
			reply.Jobs = append(reply.Jobs, *st)
		} else {
			reply.IDs = append(reply.IDs, JobStatus{ID: st.ID, Successful: st.Successful})
		}
	}

	return &reply, nil
}

// PutJob - inserts a job into the DB
func (b *Backend) PutJob(j *ProcessedJob) (*PutJobReply, error) {
	reply := PutJobReply{Status: "ok", Reason: ""}

	tx, err := b.db.Begin()
	if err != nil {
		reason := "opening SQL transaction failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}
	defer tx.Rollback()

	queryStr := "replace into Jobs values (?,?,?,?,?,?,?,?,?,?,?,?);"

	if _, err := tx.Exec(queryStr,
		j.ID, j.Repository, j.Payload, j.RepositoryPath,
		j.Script, j.ScriptArgs, j.TransferScript, strings.Join(j.Dependencies, ","),
		j.StartTime, j.FinishTime, j.Successful, j.ErrorMessage); err != nil {
		reason := "executing SQL statement failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}
	err = tx.Commit()
	if err != nil {
		reason := "committing SQL transaction failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}

	LogInfo.Printf(
		"Job inserted: %v, success: %v, start time: %v, finish time: %v\n",
		j.ID, j.Successful, j.StartTime, j.FinishTime)

	return &reply, nil
}

func scanRow(rows *sql.Rows) (*ProcessedJob, error) {
	var st ProcessedJob
	var deps string
	if err := rows.Scan(
		&st.ID, &st.Repository, &st.Payload, &st.RepositoryPath,
		&st.Script, &st.ScriptArgs, &st.TransferScript,
		&deps, &st.StartTime, &st.FinishTime,
		&st.Successful, &st.ErrorMessage); err != nil {
		return nil, err
	}
	if deps != "" {
		st.Dependencies = strings.Split(deps, ",")
	}

	return &st, nil
}

func createDataSrcName(cfg BackendConfig) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%v)/%s?parseTime=true",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}
