DROP TABLE IF EXISTS Jobs;

CREATE TABLE Jobs (
    ID char(36) NOT NULL UNIQUE PRIMARY KEY,
    Repository varchar NOT NULL,
    Payload varchar NOT NULL,
    RepositoryPath varchar NOT NULL,
    Script varchar NOT NULL,
    ScriptArgs varchar NOT NULL,
    TransferScript boolean NOT NULL,
    Dependencies varchar NOT NULL,
    StartTime timestamp NOT NULL,
    FinishTime timestamp NOT NULL,
    Successful boolean NOT NULL,
    ErrorMessage varchar NOT NULL
);

CREATE INDEX ID ON Jobs (ID);