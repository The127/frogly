-- +migrate Up

-- an update says which version it loaded by writing the next one, so the
-- database refuses it when someone saved the row in between. 40001 is
-- Postgres's code for a serialization failure, a lost race.
-- +migrate StatementBegin
create function check_version() returns trigger
    language plpgsql as
$$
begin
    if new.version <> old.version + 1 then
        raise exception '%: version mismatch', tg_table_name using errcode = '40001';
    end if;
    return new;
end;
$$;
-- +migrate StatementEnd

create table users
(
    id      uuid    not null primary key,
    version integer not null,
    name    text    not null,
    email   text    not null
);

create trigger users_version_check
    before update
    on users
    for each row
execute function check_version();

create table identities
(
    id      uuid    not null primary key,
    version integer not null,
    user_id uuid    not null references users (id),
    issuer  text    not null,
    subject text    not null,
    unique (issuer, subject)
);

create table projects
(
    id      uuid    not null primary key,
    version integer not null,
    name    text    not null
);

create trigger projects_version_check
    before update
    on projects
    for each row
execute function check_version();

create table memberships
(
    id         uuid    not null primary key,
    version    integer not null,
    project_id uuid    not null references projects (id),
    user_id    uuid    not null references users (id),
    role       text    not null,
    unique (project_id, user_id)
);

create index memberships_by_user on memberships (user_id);

-- +migrate Down

drop table memberships;
drop table projects;
drop table identities;
drop table users;
drop function check_version();
