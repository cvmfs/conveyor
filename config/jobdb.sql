DROP TABLE IF EXISTS Jobs;

CREATE TABLE Jobs (
    ID char(36) NOT NULL UNIQUE PRIMARY KEY,
    Repository varchar NOT NULL,
    Payload varchar,
    RepositoryPath varchar,
    Script varchar,
    ScriptArgs varchar,
    RemoteScript boolean,
    Dependencies varchar,
    StartTime timestamp,
    FinishTime timestamp,
    Successful boolean,
    ErrorMessage varchar
);

CREATE INDEX ID ON Jobs (ID);