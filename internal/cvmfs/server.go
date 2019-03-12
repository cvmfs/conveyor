package cvmfs

import (
	"database/sql"
	"fmt"
	"strings"

	// Import and register the PostgreSQL and MySQL drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/stdlib"
	uuid "github.com/satori/go.uuid"

	"github.com/pkg/errors"
)

const (
	// SchemaVersion is the latest schema version of the job database
	SchemaVersion = 1
)

// StartServer starts the conveyor server component. This function will block until
// the server finishes.
func StartServer(cfg *Config, keys *Keys) error {
	backend, err := startBackEnd(cfg)
	if err != nil {
		return errors.Wrap(err, "could not start service back-end")
	}
	defer backend.Close()

	if err := startFrontEnd(cfg, backend, keys); err != nil {
		return errors.Wrap(err, "could not start service front-end")
	}

	return nil
}

// serverBackend encapsulates the server state
type serverBackend struct {
	db                   *sql.DB
	dbAdapter            databaseAdapter
	pub                  *QueueClient
	newJobExchange       string
	completedJobExchange string
}

// startBackEnd initializes the backend of the job server
func startBackEnd(cfg *Config) (*serverBackend, error) {
	adapter, err := newDatabaseAdapter(cfg.Backend.Type)
	if err != nil {
		return nil, errors.Wrap(err, "could not crate database query adapter")
	}

	db, err := sql.Open(
		adapter.driverName(),
		adapter.dataSourceName(
			cfg.Backend.Username, cfg.Backend.Password, cfg.Backend.Host, cfg.Backend.Port,
			cfg.Backend.Database))

	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "connection ping failed")
	}

	currentSchemaVersion, err := getSchemaVersion(db, adapter)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve current DB schema version")
	}
	if currentSchemaVersion != SchemaVersion {
		return nil, fmt.Errorf(
			"invalid schema version: latest = %v, database = %v",
			SchemaVersion, currentSchemaVersion)
	}

	pub, err := NewQueueClient(&cfg.Queue, publisherConnection)
	if err != nil {
		return nil, errors.Wrap(err, "could not create publisher connection")
	}

	return &serverBackend{db, adapter, pub, cfg.Queue.NewJobExchange, cfg.Queue.CompletedJobExchange}, nil
}

// Close the connection to the database and the queue
func (b *serverBackend) Close() {
	b.db.Close()
	b.pub.Close()
}

// getJobStatus returns the rows from the job DB corresponding to the IDs
func (b *serverBackend) getJobStatus(ids []string, full bool) (*GetJobStatusReply, error) {
	reply := GetJobStatusReply{BasicReply: BasicReply{Status: "ok", Reason: ""}}

	queryStr := b.dbAdapter.jobStatusQuery(len(ids))
	params := make([]interface{}, len(ids))
	for i, v := range ids[0 : len(ids)-1] {
		params[i] = v
	}
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

// putNewJob publishes a new (unprocessed) job
func (b *serverBackend) putNewJob(j *JobSpecification) (*PostNewJobReply, error) {
	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate UUID")
	}

	reply := PostNewJobReply{BasicReply{Status: "ok", Reason: ""}, id}

	job := UnprocessedJob{ID: id, JobSpecification: *j}

	if err := b.pub.publish(b.newJobExchange, "", &job); err != nil {
		return nil, errors.Wrap(err, "job description publishing failed")
	}
	return &reply, nil
}

// putJobStatus inserts a job into the DB
func (b *serverBackend) putJobStatus(j *ProcessedJob) (*PostJobStatusReply, error) {
	reply := PostJobStatusReply{BasicReply{Status: "ok", Reason: ""}}

	tx, err := b.db.Begin()
	if err != nil {
		reason := "opening SQL transaction failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}
	defer tx.Rollback()

	queryStr := b.dbAdapter.insertOrUpdateJobStatement()
	if _, err := tx.Exec(queryStr,
		j.ID, j.JobName, j.Repository, j.Payload, j.RepositoryPath,
		j.Script, j.ScriptArgs, j.TransferScript, strings.Join(j.Dependencies, ","),
		j.WorkerName, j.StartTime, j.FinishTime, j.Successful, j.ErrorMessage); err != nil {
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

	status := JobStatus{ID: j.ID, Successful: j.Successful}
	var routingKey string
	if j.Successful {
		routingKey = successKey
	} else {
		routingKey = failedKey
	}
	if err := b.pub.publish(b.completedJobExchange, routingKey, &status); err != nil {
		return nil, errors.Wrap(err, "publishing job status notification failed")
	}

	Log.Info().
		Str("job_id", j.ID.String()).
		Bool("success", j.Successful).
		Time("start_time", j.StartTime).
		Time("finish_time", j.FinishTime).
		Str("worker", j.WorkerName).
		Msg("job inserted")

	return &reply, nil
}

func scanRow(rows *sql.Rows) (*ProcessedJob, error) {
	var st ProcessedJob
	var deps string
	if err := rows.Scan(
		&st.ID, &st.JobName, &st.Repository, &st.Payload, &st.RepositoryPath,
		&st.Script, &st.ScriptArgs, &st.TransferScript,
		&deps, &st.WorkerName, &st.StartTime, &st.FinishTime,
		&st.Successful, &st.ErrorMessage); err != nil {
		return nil, err
	}
	if deps != "" {
		st.Dependencies = strings.Split(deps, ",")
	}

	return &st, nil
}

func getSchemaVersion(db *sql.DB, adapter databaseAdapter) (int, error) {
	rows, err := db.Query(adapter.schemaVersionQuery())
	if err != nil {
		return 0, errors.Wrap(err, "SQL query failed")
	}
	defer rows.Close()

	maxSchemaVersion := 0
	for rows.Next() {
		var ver int
		if err := rows.Scan(&ver); err != nil {
			return 0, errors.Wrap(err, "SQL query scan failed")
		}
		if ver > maxSchemaVersion {
			maxSchemaVersion = ver
		}
	}

	return maxSchemaVersion, nil
}
