-- +migrate Up

-- the unique pair also finds a project's requests, and whether someone
-- asked to join a project
create table join_requests
(
    id         uuid        not null primary key,
    version    integer     not null,
    project_id uuid        not null references projects (id),
    user_id    uuid        not null references users (id),
    asked_at   timestamptz not null,
    unique (project_id, user_id)
);

-- +migrate Down

drop table join_requests;
