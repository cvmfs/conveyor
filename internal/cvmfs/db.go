package cvmfs

import (
	"fmt"

	"github.com/pkg/errors"
)

type databaseAdapter interface {
	driverName() string
	dataSourceName(user, pass, host string, port int, database string) string
	schemaVersionQuery() string
	jobStatusQuery(numIds int) string
	insertOrUpdateJobStatement() string
}

func newDatabaseAdapter(dbtype string) (databaseAdapter, error) {
	switch dbtype {
	case "mysql":
		return &mySQLAdapter{}, nil
	case "postgres":
		return &postgresAdapter{}, nil
	default:
		return nil, errors.New("unknown database type")
	}
}

// PostgresAdapter provides adapted queries and configuration strings for the Postgres driver:
// https://github.com/jackc/pgx
type postgresAdapter struct{}

func (a *postgresAdapter) driverName() string {
	return "pgx"
}

func (a *postgresAdapter) dataSourceName(user, pass, host string, port int, database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%v/%s?sslmode=disable", user, pass, host, port, database)
}

func (a *postgresAdapter) schemaVersionQuery() string {
	return "SELECT VersionNumber FROM SchemaVersion WHERE SchemaVersion.ValidTo IS NULL"
}

func (a *postgresAdapter) jobStatusQuery(numIds int) string {
	queryStr := "SELECT * FROM Jobs WHERE Jobs.ID IN ("
	for i := 0; i < numIds-1; i++ {
		queryStr += fmt.Sprintf("$%v, ", i+1)
	}
	queryStr += fmt.Sprintf("$%v);", numIds)
	return queryStr
}

func (a *postgresAdapter) insertOrUpdateJobStatement() string {
	return "INSERT INTO Jobs (ID, JobName, Repository, Payload, RepositoryPath, Script," +
		"ScriptArgs, TransferScript, Dependencies, WorkerName, StartTime, FinishTime, Successful, ErrorMessage) " +
		"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) " +
		"ON CONFLICT (ID) DO UPDATE " +
		"SET ID = EXCLUDED.ID, JobName = EXCLUDED.JobName, Repository = EXCLUDED.Repository, " +
		"Payload = EXCLUDED.Payload, RepositoryPath = EXCLUDED.RepositoryPath, Script = EXCLUDED.Script, " +
		"ScriptArgs = EXCLUDED.ScriptArgs, TransferScript = EXCLUDED.TransferScript, " +
		"Dependencies = EXCLUDED.Dependencies, WorkerName = EXCLUDED.WorkerName, StartTime = EXCLUDED.StartTime, " +
		"FinishTime = EXCLUDED.FinishTime, Successful = EXCLUDED.Successful, ErrorMessage = EXCLUDED.ErrorMessage;"
}

// MySQLAdapter provides adapted queries and configuration strings for the Postgres driver:
// https://github.com/go-sql-driver/mysql/
type mySQLAdapter struct{}

func (a *mySQLAdapter) driverName() string {
	return "mysql"
}

func (a *mySQLAdapter) dataSourceName(user, pass, host string, port int, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%v)/%s?parseTime=true", user, pass, host, port, database)
}

func (a *mySQLAdapter) schemaVersionQuery() string {
	return "SELECT VersionNumber FROM SchemaVersion WHERE SchemaVersion.ValidTo IS NULL"
}

func (a *mySQLAdapter) jobStatusQuery(numIds int) string {
	queryStr := "SELECT * FROM Jobs WHERE Jobs.ID IN ("
	for i := 0; i < numIds-1; i++ {
		queryStr += "?, "
	}
	queryStr += "?);"
	return queryStr
}

func (a *mySQLAdapter) insertOrUpdateJobStatement() string {
	return "REPLACE INTO Jobs VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?);"
}
