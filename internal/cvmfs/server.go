package cvmfs

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/stdlib" // Import and register the PostgreSQL driver
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
	pub                  *QueueClient
	newJobExchange       string
	completedJobExchange string
}

// startBackEnd initializes the backend of the job server
func startBackEnd(cfg *Config) (*serverBackend, error) {
	db, err := sql.Open(cfg.Backend.Type, createDataSrcName(&cfg.Backend))
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "connection ping failed")
	}

	currentSchemaVersion, err := getSchemaVersion(db)
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

	return &serverBackend{db, pub, cfg.Queue.NewJobExchange, cfg.Queue.CompletedJobExchange}, nil
}

// Close the connection to the database and the queue
func (b *serverBackend) Close() {
	b.db.Close()
	b.pub.Close()
}

// getJobStatus returns the rows from the job DB corresponding to the IDs
func (b *serverBackend) getJobStatus(ids []string, full bool) (*GetJobStatusReply, error) {
	reply := GetJobStatusReply{BasicReply: BasicReply{Status: "ok", Reason: ""}}

	queryStr := "select * from Jobs where Jobs.ID in ("
	params := make([]interface{}, len(ids))
	for i, v := range ids[0 : len(ids)-1] {
		queryStr += fmt.Sprintf("$%v, ", i+1)
		params[i] = v
	}
	queryStr += fmt.Sprintf("$%v);", len(ids))
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

	row := tx.QueryRow("select ID from Jobs where ID == $1;", j.ID)
	if err != nil {
		reason := "SQL query failed"
		reply.Status = "error"
		reply.Reason = reason
		return &reply, errors.Wrap(err, reason)
	}

	var queryStr string
	var i uuid.UUID
	if row.Scan(&i) == sql.ErrNoRows {
		queryStr = "insert into Jobs (ID,JobName,Repository,Payload,RepositoryPath,Script,ScriptArgs," +
			"TransferScript,Dependencies,WorkerName,StartTime,FinishTime,Successful,ErrorMessage) " +
			"values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14);"
	} else {
		queryStr = "update Jobs set (ID = $1, JobName = $2, Repository = $3, Payload = $4," +
			"RepositoryPath = $5, Script = $6, ScriptArgs = $7, TransferScript = $8, Dependencies = $9," +
			"WorkerName = $10, StartTime = $11, FinishTime = $12, Successful = $13, ErrorMessage = $14) " +
			"where Jobs.ID == $1;"
	}

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

func createDataSrcName(cfg *BackendConfig) string {
	// return fmt.Sprintf(
	// 	"%s:%s@tcp(%s:%v)/%s?parseTime=true",
	// 	cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%v/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}

func getSchemaVersion(db *sql.DB) (int, error) {
	rows, err := db.Query(
		"select VersionNumber from SchemaVersion where SchemaVersion.ValidTo is NULL")
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
