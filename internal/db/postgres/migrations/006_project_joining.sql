-- +migrate Up

-- every public project so far could only be joined through a member
alter table projects
    add column joining text not null default 'closed'
        check (joining in ('open', 'closed'));

-- +migrate Down

alter table projects drop column joining;
