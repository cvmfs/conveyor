USE devcvmfs;

CREATE TABLE IF NOT EXISTS Jobs (
    ID char(36) NOT NULL UNIQUE PRIMARY KEY,
    Repository varchar(65535) NOT NULL,
    Payload varchar(65535) NOT NULL,
    RepositoryPath varchar(65535) NOT NULL,
    Script varchar(65535) NOT NULL,
    ScriptArgs varchar(65535) NOT NULL,
    TransferScript boolean NOT NULL,
    Dependencies varchar(65535) NOT NULL,
    StartTime timestamp NOT NULL,
    FinishTime timestamp NOT NULL,
    Successful boolean NOT NULL,
    ErrorMessage varchar(65535) NOT NULL
);
