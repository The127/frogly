-- +migrate Up

create table users
(
    id      text    not null primary key,
    version integer not null,
    name    text    not null,
    email   text    not null
);

-- an update says which version it loaded by writing the next one, so the
-- database refuses it when someone saved the row in between
-- +migrate StatementBegin
create trigger users_version_check
    before update
    on users
    for each row
    when new.version <> old.version + 1
begin
    select raise(fail, 'users: version mismatch');
end;
-- +migrate StatementEnd

create table identities
(
    id      text    not null primary key,
    version integer not null,
    user_id text    not null references users (id),
    issuer  text    not null,
    subject text    not null,
    unique (issuer, subject)
);

create table projects
(
    id      text    not null primary key,
    version integer not null,
    name    text    not null
);

-- +migrate StatementBegin
create trigger projects_version_check
    before update
    on projects
    for each row
    when new.version <> old.version + 1
begin
    select raise(fail, 'projects: version mismatch');
end;
-- +migrate StatementEnd

create table memberships
(
    id         text    not null primary key,
    version    integer not null,
    project_id text    not null references projects (id),
    user_id    text    not null references users (id),
    role       text    not null,
    unique (project_id, user_id)
);

create index memberships_by_user on memberships (user_id);

-- +migrate Down

drop table memberships;
drop table projects;
drop table identities;
drop table users;
