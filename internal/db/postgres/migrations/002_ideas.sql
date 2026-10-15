-- +migrate Up

create table ideas
(
    id         uuid        not null primary key,
    version    integer     not null,
    project_id uuid        not null references projects (id),
    planter_id uuid        not null references users (id),
    text       text        not null,
    planted_at timestamptz not null
);

create trigger ideas_version_check
    before update
    on ideas
    for each row
execute function check_version();

create index ideas_by_project on ideas (project_id, planted_at desc, id desc);

-- +migrate Down

drop table ideas;
