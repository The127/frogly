-- +migrate Up

-- every public project so far could only be joined through a member
alter table projects
    add column joining text not null default 'closed'
        check (joining in ('open', 'closed'));

-- nobody outside can see a private project, so it's always closed
alter table projects
    add constraint projects_private_is_closed check (joining = 'closed' or visibility = 'public');

-- +migrate Down

alter table projects drop constraint projects_private_is_closed;
alter table projects drop column joining;
