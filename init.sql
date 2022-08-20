CREATE TABLE users (
    id bigint UNIQUE,
    email text,
    password text,
    name text,
    username text UNIQUE,
    icon text
);
CREATE TABLE lists (
    id bigint UNIQUE,
    user_id bigint,
    name text,
    comment text DEFAULT '',
    index integer
);
CREATE TABLE tasks (
    id bigint UNIQUE,
    list_id bigint,
    task_id bigint,
    name text,
    comment text DEFAULT '',
    index integer,
    categories text [],
    end_time timestamptz DEFAULT null,
    done boolean DEFAULT false,
    special boolean DEFAULT false
);
