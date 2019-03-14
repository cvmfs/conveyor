CREATE TABLE IF NOT EXISTS SchemaVersion (
    VersionNumber int NOT NULL UNIQUE PRIMARY KEY,
    ValidFrom timestamp NOT NULL,
    ValidTo timestamp
);

INSERT INTO SchemaVersion (VersionNumber, ValidFrom)
    VALUES (1, NOW());

CREATE TABLE IF NOT EXISTS Jobs (
    ID char(36) NOT NULL UNIQUE PRIMARY KEY,
    JobName varchar(65535) NOT NULL,
    Repository varchar(65535) NOT NULL,
    Payload varchar(65535) NOT NULL,
    LeasePath varchar(65535) NOT NULL,
    Dependencies varchar(65535) NOT NULL,
    WorkerName varchar(65535) NOT NULL,
    StartTime timestamp NOT NULL,
    FinishTime timestamp NOT NULL,
    Successful boolean NOT NULL,
    ErrorMessage varchar(65535) NOT NULL
);
