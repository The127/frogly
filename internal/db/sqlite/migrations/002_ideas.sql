-- +migrate Up

-- planted_at is microseconds since 1970 in UTC, since SQLite has no type
-- for time, and a number sorts the way time does
create table ideas
(
    id         text    not null primary key,
    version    integer not null,
    project_id text    not null references projects (id),
    planter_id text    not null references users (id),
    text       text    not null,
    planted_at integer not null
);

-- +migrate StatementBegin
create trigger ideas_version_check
    before update
    on ideas
    for each row
    when new.version <> old.version + 1
begin
    select raise(fail, 'ideas: version mismatch');
end;
-- +migrate StatementEnd

create index ideas_by_project on ideas (project_id, planted_at desc, id desc);

-- +migrate Down

drop table ideas;
