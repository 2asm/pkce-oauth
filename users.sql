DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id serial PRIMARY KEY,
    name varchar(30) NOT NULL,
    email varchar(30) NOT NULL,
    password varchar(50),
    created_at timestamptz,    
    updated_at timestamptz 
);

-- insert into users (id, name, created_at, updated_at) values ('bbb0206c-b55c-11ee-8e0d-8bb83298bd36', 'dilip kumar', now(), now());
