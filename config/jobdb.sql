DROP TABLE IF EXISTS jobs;

CREATE TABLE jobs (
    id char(36) NOT NULL UNIQUE PRIMARY KEY,
    repo varchar NOT NULL,
    payload varchar,
    repo_path varchar,
    script varchar,
    script_args varchar,
    remote_script boolean,
    dependencies varchar,
    start_time timestamp,
    finish_time timestamp,
    status varchar
);

CREATE INDEX id ON jobs (id);